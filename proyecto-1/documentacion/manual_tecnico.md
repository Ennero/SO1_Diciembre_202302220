# Manual Técnico — Proyecto 1 (SO1)

Sistema de monitoreo de contenedores que integra módulos de Kernel en C y un daemon en Go para recolectar métricas, cruzarlas con Docker y gestionar el ciclo de vida de procesos/containers de alto o bajo consumo.

## Índice

## Índice

- [Manual Técnico — Proyecto 1 (SO1)](#manual-técnico--proyecto-1-so1)
	- [Índice](#índice)
	- [Índice](#índice-1)
	- [1. Arquitectura del Sistema](#1-arquitectura-del-sistema)
	- [2. Módulo de Kernel (C)](#2-módulo-de-kernel-c)
		- [2.1. Archivos y Funciones](#21-archivos-y-funciones)
		- [2.2. JSON Expuesto](#22-json-expuesto)
	- [3. Daemon (Go)](#3-daemon-go)
		- [3.1. Funcionalidades Clave](#31-funcionalidades-clave)
		- [3.2. Lógica de Control de Contenedores](#32-lógica-de-control-de-contenedores)
	- [4. Automatización y Cronjob](#4-automatización-y-cronjob)
	- [5. Base de Datos y Persistencia](#5-base-de-datos-y-persistencia)
	- [6. Visualización en Grafana](#6-visualización-en-grafana)
		- [6.1. Dashboard 1: Contenedores](#61-dashboard-1-contenedores)
		- [6.2. Dashboard 2: Sistema](#62-dashboard-2-sistema)
	- [7. Instalación y Ejecución](#7-instalación-y-ejecución)
		- [Requisitos Previos](#requisitos-previos)
		- [Paso 1: Construir Imágenes Docker](#paso-1-construir-imágenes-docker)
		- [Paso 2: Compilar Módulos](#paso-2-compilar-módulos)
		- [Paso 3: Ejecutar Daemon](#paso-3-ejecutar-daemon)
		- [Paso 4: Ver Resultados](#paso-4-ver-resultados)
	- [8. Decisiones de Diseño y Problemas Encontrados](#8-decisiones-de-diseño-y-problemas-encontrados)
		- [8.1. Decisiones de Diseño](#81-decisiones-de-diseño)
		- [8.2. Problemas y Soluciones](#82-problemas-y-soluciones)
	- [9. Verificación Manual (Debugging)](#9-verificación-manual-debugging)
	- [10. Carpeta Compartida Host↔VM (Virtio-FS)](#10-carpeta-compartida-hostvm-virtio-fs)
		- [Migrar el proyecto desde carpeta compartida a Home](#migrar-el-proyecto-desde-carpeta-compartida-a-home)

## 1. Arquitectura del Sistema

El proyecto sigue una arquitectura de monitoreo desacoplada:

- **Nivel Kernel (C)**: Dos módulos (`sysinfo.ko` y `continfo.ko`) acceden a las estructuras internas de Linux (`task_struct`, `sysinfo`) y exponen los datos en formato JSON a través del sistema de archivos virtual `/proc`.
- **Nivel Usuario (Go Daemon)**: Un servicio persistente lee los archivos `/proc`, procesa la información, gestiona la base de datos SQLite y ejecuta comandos de sistema (Docker, Cron, Kill).
- **Nivel Visualización (Grafana)**: Un contenedor Docker lee la base de datos SQLite y muestra métricas en tiempo real.

## 2. Módulo de Kernel (C)

**Ubicación**: `modulo-kernel/`  
**Archivos fuente**: `sysinfo.c` y `continfo.c`  
**Dependencias**: Headers de Linux (`linux/module.h`, `linux/sched.h`, `linux/mm.h`, etc.)

### 2.1. Archivos y Funciones

| Módulo   | Archivo Fuente | Archivo Generado | Ruta /proc                     | Descripción                                                                                     |
| -------- | -------------- | ---------------- | ------------------------------ | ----------------------------------------------------------------------------------------------- |
| Procesos | `sysinfo.c`    | `sysinfo.ko`     | `/proc/sysinfo_so1_202302220`  | Itera sobre `for_each_process`, recolecta PID, Nombre, Estado, RAM (RSS), VSZ y tiempos de CPU. |
| Memoria  | `continfo.c`   | `continfo.ko`    | `/proc/continfo_so1_202302220` | Utiliza `si_meminfo` para obtener métricas globales de RAM (Total, Libre, Usada).               |

### 2.2. JSON Expuesto

**Salida de sysinfo (Array de Procesos)**:
```json
[
  {
    "pid": 1234,
    "name": "stress-ng",
    "state": 0,
    "ram_kb": 25600,
    "vsz_kb": 50000,
    "ram_percent": 2,
    "cpu_utime": 100,
    "cpu_stime": 50
  },
  ...
]
```

**Salida de continfo (Objeto Global)**:
```json
{
  "total_ram_mb": 8000,
  "free_ram_mb": 4000,
  "used_ram_mb": 3500,
  "percentage": 43
}
```

## 3. Daemon (Go)

**Ubicación**: `go-daemon/main.go`  
**Rol**: Orquestador central del proyecto.

### 3.1. Funcionalidades Clave

- **Carga de Módulos**: Al iniciar, ejecuta automáticamente `insmod` para cargar `sysinfo.ko` y `continfo.ko`.
- **Configuración de Cronjob**: Inyecta en el crontab del sistema la ejecución de `generator.sh` cada minuto.
- **Despliegue de Grafana**: Ejecuta `docker run` para levantar Grafana en el puerto 3000, montando la base de datos `metrics.db` y un volumen persistente para dashboards.
- **Loop de Monitoreo (20s)**:
  - Lee `/proc/sysinfo...` y `/proc/continfo...`.
  - Calcula el porcentaje de CPU comparando el tiempo actual con el historial previo (`delta_cpu / delta_time`).
  - Guarda métricas en SQLite.
  - Ejecuta la lógica de control de contenedores.

### 3.2. Lógica de Control de Contenedores

El sistema mantiene un equilibrio estricto de contenedores definidos en las constantes:

- `DESIRED_LOW = 3` (Contenedores tipo `sleep`)
- `DESIRED_HIGH = 2` (Contenedores tipo `stress-ng`)

**Algoritmo de Matanza**:

Si el conteo detectado supera el deseado:
1. Identifica los PIDs excedentes.
2. Prioriza matar al proceso padre (nombre exacto `stress-ng` o `sleep`).
3. Ejecuta `kill -9 <PID>`.
4. Registra la acción en la tabla `kill_log`.

## 4. Automatización y Cronjob

**Script**: `bash/generator.sh`  
**Función**: Genera 10 contenedores aleatorios seleccionando entre las imágenes `so1_ram`, `so1_cpu` y `so1_low`.

**Ejecución**: Gestionada por el Daemon de Go, que añade la siguiente línea al crontab:
```bash
* * * * * /bin/bash /ruta/absoluta/a/bash/generator.sh
```

**Limpieza**: Al detener el daemon (Ctrl+C), se captura la señal SIGINT y se elimina esta línea del crontab automáticamente.

## 5. Base de Datos y Persistencia

**Motor**: SQLite 3  
**Archivo**: `go-daemon/metrics.db`

**Esquema**:
- `ram_log`: Histórico de memoria global.
- `process_log`: Snapshot de todos los procesos (PID, nombre, RAM, CPU) cada 20s.
- `kill_log`: Registro de auditoría de contenedores eliminados por el daemon.

## 6. Visualización en Grafana

Se implementan dos dashboards conectados a `metrics.db`.

### 6.1. Dashboard 1: Contenedores

Filtra métricas solo para procesos `stress-ng` y `sleep`.

- **Total RAM**: `SELECT total FROM ram_log ORDER BY id DESC LIMIT 1;`
- **Free RAM**: `SELECT (total - used) FROM ram_log ORDER BY id DESC LIMIT 1;`
- **Contenedores Eliminados**: `SELECT timestamp as time, count(id) as value FROM kill_log ...`
- **Uso de RAM (Tiempo)**: `SELECT timestamp as time, used FROM ram_log ...`
- **Top 5 RAM (Pie)**: `SELECT name || ' (' || pid || ')', MAX(ram) FROM process_log WHERE name LIKE 'stress%' OR name = 'sleep' ...`
- **Top 5 CPU (Pie)**: Similar al anterior pero con campo `cpu`.
- **RAM Usada (Stat)**: Valor actual de memoria usada.
- **Contenedores Activos (Extra)**: `SELECT timestamp, count(distinct pid) FROM process_log ...`

### 6.2. Dashboard 2: Sistema

Muestra métricas generales de todos los procesos del SO.

- **Total RAM** (Igual al anterior).
- **Free RAM** (Igual al anterior).
- **Total Procesos**: `SELECT count(distinct pid) FROM process_log WHERE timestamp = (SELECT MAX(timestamp)...)`
- **Uso RAM Tiempo** (Igual al anterior).
- **Top 5 Sistema RAM**: `SELECT name || ' (' || pid || ')', MAX(ram) FROM process_log GROUP BY pid ...` (Sin filtro WHERE de nombre).
- **Top 5 Sistema CPU**: Igual al anterior con campo `cpu`.
- **RAM Usada** (Igual al anterior).
- **Carga Promedio CPU (Extra)**: `SELECT timestamp, avg(cpu) FROM process_log ...`

## 7. Instalación y Ejecución

### Requisitos Previos

- Linux (Kernel 5.x+ recomendado)
- Docker instalado
- Go 1.20+
- `build-essential` (Make, GCC)

### Paso 1: Construir Imágenes Docker
```bash
cd proyecto-1
docker build -t so1_ram -f docker-files/dockerfile.ram .
docker build -t so1_cpu -f docker-files/dockerfile.cpu .
docker build -t so1_low -f docker-files/dockerfile.low .
```

### Paso 2: Compilar Módulos
```bash
cd modulo-kernel
make clean && make
# Esto genera sysinfo.ko y continfo.ko
```

### Paso 3: Ejecutar Daemon

Este paso carga los módulos, configura el cron, levanta Grafana y empieza el monitoreo.
```bash
cd ../go-daemon
go mod tidy
sudo go run main.go
```

### Paso 4: Ver Resultados

- **Grafana**: Acceder a http://localhost:3000 (admin/admin).
- **Logs**: Ver la terminal donde corre el daemon para ver las acciones de eliminación.
- **Sistema**: Verificar `/proc/sysinfo_so1_...` para ver el JSON crudo.


## 8. Decisiones de Diseño y Problemas Encontrados

### 8.1. Decisiones de Diseño
- **Daemon en Go vs Script Bash:** Se eligió Go por su capacidad de manejo de concurrencia (goroutines) para correr el monitor y el generador de tráfico simultáneamente sin bloquearse, además de su facilidad para interactuar con SQLite.
- **SQLite vs InfluxDB:** Se optó por SQLite por simplicidad de despliegue (archivo único) y persistencia ligera, cumpliendo con los requisitos sin la sobrecarga de una base de datos de series temporales completa.
- **Virtio-FS:** Se seleccionó sobre las carpetas compartidas tradicionales de VirtualBox para mejorar la velocidad de I/O al compilar en la VM.

### 8.2. Problemas y Soluciones
- **Problema:** *Error "Invalid Parameters" al insertar el módulo.*
  - **Causa:** Intentar crear archivos en `/proc` con permisos de escritura sin definir handlers.
  - **Solución:** Se establecieron permisos `0444` (solo lectura) en `proc_create`.
- **Problema:** *Incompatibilidad de `task->state`.*
  - **Causa:** Cambios en versiones recientes del kernel (5.14+).
  - **Solución:** Se utilizó `task->__state` y el especificador de formato `%u`.
- **Problema:** *Procesos Zombie del Kernel.*
  - **Causa:** Errores de punteros en el módulo C colgaban el sistema.
  - **Solución:** Se implementó limpieza rigurosa en `my_module_exit` y validación de `task->mm` antes de leer memoria.

## 9. Verificación Manual (Debugging)

Aunque el Daemon gestiona los módulos, para depuración manual se pueden usar:

```bash
# Verificar que los módulos están cargados
lsmod | grep so1

# Ver logs del kernel (útil para ver printk)
dmesg | tail -n 20

```


## 10. Carpeta Compartida Host↔VM (Virtio-FS)

Este método recomendado permite montar una carpeta de tu PC dentro de la VM como si fuera otro disco. Es rápido y eficiente para uso frecuente.

Pasos en el Host (Virt-Manager):
- Abre Virtual Machine Manager (`virt-manager`).
- Abre tu VM y haz clic en el ícono de la bombilla (Detalles de hardware).
- Haz clic en "Añadir Hardware" → "Sistema de archivos" (Filesystem).
- Configura:
	- Controlador (Driver): `virtiofs` (o `virtio-fs`).
	- Ruta fuente (Source path): carpeta en tu PC (ej. `/home/tu_usuario/Compartido`).
	- Ruta destino (Target path): nombre identificador (ej. `micarpeta`).
- Finaliza y enciende la VM.

Pasos dentro de la VM (Linux):

```bash
# Crear el punto de montaje
sudo mkdir -p /mnt/compartido

# Montar la carpeta (usa el nombre del Target path configurado)
sudo mount -t virtiofs micarpeta /mnt/compartido
```

Opcional (montaje persistente al arranque):

```bash
echo 'micarpeta /mnt/compartido virtiofs defaults 0 0' | sudo tee -a /etc/fstab
sudo mount -a
```

Nota: Si tu entorno no soporta `virtio-fs`, puedes usar 9p como alternativa:

```bash
sudo mount -t 9p -o trans=virtio,version=9p2000.L micarpeta /mnt/compartido
```

### Migrar el proyecto desde carpeta compartida a Home 

Para evitar problemas de permisos y rendimiento, se recomienda copiar el proyecto desde la carpeta compartida a tu `Home` dentro de la VM y trabajar desde ahí:

```bash
# 1. Ir a tu carpeta personal (Home)
cd ~

# 2. Copiar todo el proyecto desde la carpeta compartida hacia aquí
cp -r /mnt/compartido/proyecto-1 .

# 3. Entrar a la nueva copia (que ya es 100% Linux)
cd proyecto-1
```
