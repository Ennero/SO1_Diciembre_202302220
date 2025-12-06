# Manual Técnico — Proyecto 1 (SO1)

Sistema de monitoreo de contenedores que integra un módulo de Kernel en C y un daemon en Go para recolectar métricas, cruzarlas con Docker y gestionar el ciclo de vida de procesos/containers de alto o bajo consumo.

## Índice

- [Manual Técnico — Proyecto 1 (SO1)](#manual-técnico--proyecto-1-so1)
	- [Índice](#índice)
	- [1. Arquitectura del Sistema](#1-arquitectura-del-sistema)
	- [2. Módulo de Kernel (C)](#2-módulo-de-kernel-c)
		- [2.1. Funciones principales](#21-funciones-principales)
		- [2.2. Métricas expuestas](#22-métricas-expuestas)
	- [3. Daemon (Go)](#3-daemon-go)
		- [3.1. Cálculo de %CPU](#31-cálculo-de-cpu)
		- [3.2. Política de control](#32-política-de-control)
		- [3.3. Automatizaciones y limpieza](#33-automatizaciones-y-limpieza)
	- [4. Automatización (Bash)](#4-automatización-bash)
	- [5. Decisiones de Diseño y Problemas Encontrados](#5-decisiones-de-diseño-y-problemas-encontrados)
		- [Problema 1: Incompatibilidad de `task->state`](#problema-1-incompatibilidad-de-task-state)
		- [Problema 2: Error “Invalid Parameters” al insertar el módulo](#problema-2-error-invalid-parameters-al-insertar-el-módulo)
		- [Problema 3: Módulo “zombie”](#problema-3-módulo-zombie)
	- [6. Instalación y Ejecución](#6-instalación-y-ejecución)
		- [6.1. Requisitos Previos](#61-requisitos-previos)
		- [6.2. Construcción de Imágenes Docker](#62-construcción-de-imágenes-docker)
		- [6.3. Compilación y Carga de los Módulos](#63-compilación-y-carga-de-los-módulos)
		- [6.4. Generación de Tráfico](#64-generación-de-tráfico)
		- [6.5. Iniciar el Monitor (Daemon)](#65-iniciar-el-monitor-daemon)
		- [6.6. Carpeta Compartida Host↔VM (Virtio-FS)](#66-carpeta-compartida-hostvm-virtio-fs)
		- [6.6.1. Migrar proyecto desde carpeta compartida a Home (VM)](#661-migrar-proyecto-desde-carpeta-compartida-a-home-vm)
		- [6.7. Levantar Grafana](#67-levantar-grafana)
	- [7. Notas de Seguridad y Mantenimiento](#7-notas-de-seguridad-y-mantenimiento)
	- [8. Referencias Rápidas](#8-referencias-rápidas)

## 1. Arquitectura del Sistema

- **Espacio de Kernel (C):** Módulos `procesos.ko` y `ram.ko` exponen métricas en `/proc/sysinfo_so1_202302220` (procesos) y `/proc/continfo_so1_202302220` (RAM) en JSON.
- **Espacio de Usuario (Go):** Daemon lee ambos `/proc`, calcula %CPU por diferencia, registra en SQLite (`metrics.db`), levanta Grafana con volúmenes persistentes y aplica política de control sobre contenedores/procesos stress/sleep.

## 2. Módulo de Kernel (C)

- **Ubicación:** `proyecto-1/modulo-kernel` — [abrir carpeta](../modulo-kernel/)
- **Archivos:** `procesos.c` (procesos) y `ram.c` (memoria)
- **Procfs expuestos:** `/proc/sysinfo_so1_202302220` y `/proc/continfo_so1_202302220`
- **Dependencias (headers):** `<linux/module.h>`, `<linux/sched.h>`, `<linux/mm.h>`, `<linux/seq_file.h>`, `<linux/sched/signal.h>`, `<linux/sysinfo.h>`

### 2.1. Funciones principales

- **`my_module_init` (ambos):** Crea entradas en `/proc` con permisos `0444` (solo lectura).
- **`my_proc_show` en `procesos.c`:** Recorre procesos (`for_each_process`) y expone arreglo JSON con `pid`, `name`, `state`, `ram_kb` (RSS), `vsz_kb` (VSZ), `cpu_utime`, `cpu_stime`.
- **`my_proc_show` en `ram.c`:** Expone objeto JSON con `total_ram_mb`, `free_ram_mb`, `used_ram_mb`, `percentage`.
- **`my_module_exit` (ambos):** Elimina la entrada en `/proc`.

### 2.2. Métricas expuestas

Cada entrada del arreglo JSON tiene la forma:

```
{
	"pid": <int>,
	"name": "<string>",
	"state": <uint>,
	"ram_kb": <ulong>,
	"vsz_kb": <ulong>,
	"cpu_utime": <ull>,
	"cpu_stime": <ull>
}
```

## 3. Daemon (Go)

- **Ubicación:** `proyecto-1/go-daemon/main.go` — [ver archivo](../go-daemon/main.go)
- **Frecuencia:** Ticker cada 5 segundos.
- **Tareas por ciclo:**
  - Obtener contenedores con `docker ps` (`exec.Command`).
	- Leer y decodificar JSON de `/proc/sysinfo_so1_202302220` (procesos) y `/proc/continfo_so1_202302220` (RAM).
  - Calcular `%CPU` por diferencia de tiempos acumulados (`utime + stime`).
  - Clasificar procesos en “ALTO” y “BAJO” consumo y aplicar límites.

### 3.1. Cálculo de %CPU

Se calcula por diferencia, asumiendo que `cpu_utime` + `cpu_stime` están en nanosegundos acumulados:

$$
\%\,CPU = \frac{\Delta(utime + stime)}{\Delta t_{real}} \times 100
$$

Implementación: `(deltaCpu / deltaTime.Nanoseconds()) * 100`. Se filtran valores <0, NaN/Inf y se capea en 400% para evitar outliers. El daemon guarda un historial por PID para el cálculo incremental.

### 3.2. Política de control 

- **Constantes:** `DESIRED_HIGH = 2`, `DESIRED_LOW = 3`.
- **Clasificación:** Procesos cuyo nombre contiene `stress` → ALTO; contienen `sleep` → BAJO. Para “altos” solo se cuenta como contenedor el padre `stress-ng`; los hijos (`stress-ng-cpu`, `stress-ng-vm`) se loguean y pueden ser eliminados si sobra.
- **Acción:** Si exceden los deseados, se mata primero al padre objetivo (`stress-ng` para altos, `sleep` para bajos) y luego hijos huérfanos si aún falta por ajustar. Se registran muertes en `kill_log`.

### 3.3. Automatizaciones y limpieza

- **Generador de tráfico:** `generator.sh` se ejecuta cada 60s desde el daemon (goroutine) para crear contenedores de prueba.
- **Manejo de señales:** `setupSignalHandler` captura `SIGINT`/`SIGTERM`; al salir limpia la crontab removiendo cualquier línea que contenga `bash/generator.sh`.

## 4. Automatización (Bash)

- **Ubicación:** `proyecto-1/bash/generator.sh` — [ver archivo](../bash/generator.sh)
- **Función:** Estresar el sistema para pruebas creando 10 contenedores aleatorios basados en las imágenes `so1_ram`, `so1_cpu`, `so1_low`. Nombres únicos para facilitar rastreo. El daemon lo invoca automáticamente cada 60s y limpia la crontab al salir.

## 5. Decisiones de Diseño y Problemas Encontrados

### Problema 1: Incompatibilidad de `task->state`
**Solución:** Reinicio de la VM para limpiar estado del kernel y build limpio (asegurar nombre de objeto correcto). El Makefile genera `procesos.ko` y `ram.ko`.
- **Solución:** Usar `task->__state` y ajustar formato de impresión a unsigned (`%u`).

### Problema 2: Error “Invalid Parameters” al insertar el módulo
- **Descripción:** `insmod` rechazaba el módulo.
- **Causa:** Se intentó crear `/proc` con permisos `0777` sin handlers de escritura.
- **Solución:** Permisos `0444` (solo lectura) en `proc_create`, consistente con uso informativo.

### Problema 3: Módulo “zombie”
- **Descripción:** Tras un fallo, no se podía descargar (`rmmod`) ni recargar.
- **Solución:** Reinicio de la VM para limpiar estado del kernel y build limpio (asegurar nombre de objeto correcto). El Makefile genera `module.ko`.

## 6. Instalación y Ejecución

### 6.1. Requisitos Previos

- **SO:** Linux (Ubuntu 22.04+ recomendado)
- **Docker:** Instalado y servicio activo
- **Go:** v1.20+
- **Herramientas:** GCC y Make

Instalar todas las dependencias necesarias (Ubuntu/Debian)

```bash
# Actualizar índices
sudo apt update

# Herramientas de compilación y headers del kernel (para construir .ko)
sudo apt install -y build-essential linux-headers-$(uname -r)

# Docker y Docker Compose (usa docker-compose v1)
sudo apt install -y docker.io docker-compose

# Alternativa: instalar Docker vía Snap (si tu entorno lo prefiere)
sudo snap install docker

# Habilitar y arrancar Docker
sudo systemctl enable --now docker

# Permitir usar Docker sin sudo (opcional)
sudo usermod -aG docker "$USER"
echo "[INFO] Cierra sesión y vuelve a entrar para aplicar grupo docker"

# Go (si no lo tienes). Alternativa: usar versión del repositorio
sudo apt install -y golang

# Verificaciones rápidas
docker --version
docker-compose --version
go version
gcc --version
make --version
```

Iniciar y verificar el servicio Docker

```bash
# Iniciar el servicio Docker
sudo systemctl start docker

# Verificar estado (debe decir "Active: active (running)")
sudo systemctl status docker
```

### 6.2. Construcción de Imágenes Docker

Ejecute desde la raíz del repo o entrando a `proyecto-1`:

```bash
cd proyecto-1
docker build -t so1_ram -f docker-files/dockerfile.ram .
docker build -t so1_cpu -f docker-files/dockerfile.cpu .
docker build -t so1_low -f docker-files/dockerfile.low .
```

Archivos Dockerfiles:

- `docker-files/dockerfile.ram` — [ver archivo](../docker-files/dockerfile.ram)
- `docker-files/dockerfile.cpu` — [ver archivo](../docker-files/dockerfile.cpu)
- `docker-files/dockerfile.low` — [ver archivo](../docker-files/dockerfile.low)

### 6.3. Compilación y Carga de los Módulos

```bash
cd modulo-kernel/
make clean && make
sudo insmod procesos.ko
sudo insmod ram.ko

# Verificación
cat /proc/sysinfo_so1_202302220
cat /proc/continfo_so1_202302220
```

Archivos relacionados:

- `modulo-kernel/Makefile` — [ver archivo](../modulo-kernel/Makefile)
- `modulo-kernel/procesos.c` — [ver archivo](../modulo-kernel/procesos.c)
- `modulo-kernel/ram.c` — [ver archivo](../modulo-kernel/ram.c)

### 6.4. Generación de Tráfico

```bash
cd ../bash
chmod +x generator.sh
./generator.sh
```

Archivo relacionado: `bash/generator.sh` — [ver archivo](../bash/generator.sh)

### 6.5. Iniciar el Monitor (Daemon)

```bash
cd ../go-daemon
# Instalar dependencias de Go (una vez)
go mod tidy

sudo env "PATH=$PATH" go run main.go
```

Archivo relacionado: `go-daemon/main.go` — [ver archivo](../go-daemon/main.go)

**Resultado esperado:**
- Cada 5 segundos: RAM, procesos stress/sleep, %CPU (por diferencia), conteo y posibles kills.
- Cada 60 segundos: se ejecuta `generator.sh` para generar carga.
- Grafana se levanta automáticamente vía `docker run` con volúmenes:
	- `go-daemon/metrics.db` → `/var/lib/grafana/metrics.db` (ro)
	- `dashboard/grafana_data` → `/var/lib/grafana` (rw, persistente)
- Al recibir `Ctrl+C`/`SIGTERM`: limpia entradas de crontab que invoquen `bash/generator.sh`.

### 6.6. Carpeta Compartida Host↔VM (Virtio-FS)

Para compartir archivos con la VM de forma eficiente (ej. intercambiar `metrics.db`, logs, etc.), se recomienda montar una carpeta del Host usando `virtio-fs`:

1) En el Host (Virt-Manager):
- VM → "Mostrar detalles del hardware" (ícono bombilla) → "Añadir Hardware" → "Sistema de archivos".
- Controlador: `virtiofs`.
- `Source path`: carpeta del Host (ej. `/home/tu_usuario/Compartido`).
- `Target path`: nombre identificador (ej. `micarpeta`).

2) En la VM (Linux):
```bash
sudo mkdir -p /mnt/compartido
sudo mount -t virtiofs micarpeta /mnt/compartido
```

Persistencia opcional:
```bash
echo 'micarpeta /mnt/compartido virtiofs defaults 0 0' | sudo tee -a /etc/fstab
sudo mount -a
```

Alternativa (si `virtio-fs` no está disponible):
```bash
sudo mount -t 9p -o trans=virtio,version=9p2000.L micarpeta /mnt/compartido
```

### 6.6.1. Migrar proyecto desde carpeta compartida a Home (VM)

Para evitar incompatibilidades de permisos/ACL y mejorar rendimiento de IO, copia el proyecto a tu Home dentro de la VM y trabaja desde ahí:

```bash
# 1. Ir a tu carpeta personal (Home)
cd ~

# 2. Copiar todo el proyecto desde la carpeta compartida hacia aquí
cp -r /mnt/compartido/proyecto-1 .

# 3. Entrar a la nueva copia (nativa de Linux)
cd proyecto-1

```

### 6.7. Levantar Grafana

El daemon ya levanta Grafana con `docker run` y volúmenes persistentes. Preparación mínima:

```bash
cd proyecto-1
touch go-daemon/metrics.db
chmod 666 go-daemon/metrics.db
mkdir -p dashboard/grafana_data && chmod 777 dashboard/grafana_data
# Luego ejecutar el daemon (6.5). Grafana quedará arriba en :3000
```

Acceso: `http://localhost:3000` (Usuario: `admin` / Password: `admin`).

Persistencia: dashboards, datasources, usuarios en `dashboard/grafana_data`; métricas en `go-daemon/metrics.db` montada de solo lectura.

¿Usar docker-compose? Sigue soportado (`dashboard/docker-compose.yml`) y usa los mismos volúmenes.

Consultas para Dashboard (8 vistas):

1) Monitor de RAM en Tiempo Real (Dientes de Sierra) — Time Series

```sql
SELECT 
	timestamp,
	percentage AS "Uso RAM %"
FROM ram_log 
ORDER BY timestamp ASC;
```

2) Consumo de RAM en MB (Actual) — Gauge/Stat

```sql
SELECT 
	used AS "MB Usados"
FROM ram_log 
ORDER BY timestamp DESC 
LIMIT 1;
```

3) Total de Contenedores Eliminados — Stat

```sql
SELECT COUNT(*) AS "Total Eliminados" FROM kill_log;
```

4) Historial de Eliminaciones (Log de Muertes) — Table

```sql
SELECT 
	timestamp AS "Fecha/Hora", 
	pid AS "PID", 
	name AS "Nombre Contenedor", 
	reason AS "Razón"
FROM kill_log 
ORDER BY timestamp DESC;
```

5) Top Contenedores por Consumo de CPU (Histórico) — Bar Gauge

```sql
SELECT 
	name || ' (' || pid || ')' AS Container, 
	MAX(cpu) AS "Max CPU %"
FROM process_log 
GROUP BY pid, name
ORDER BY "Max CPU %" DESC 
LIMIT 5;
```

6) Top Contenedores por Consumo de RAM (Histórico) — Bar Gauge

```sql
SELECT 
	name || ' (' || pid || ')' AS Container, 
	MAX(ram) AS "Max RAM MB"
FROM process_log 
GROUP BY pid, name
ORDER BY "Max RAM MB" DESC 
LIMIT 5;
```

7) Tabla de Procesos en Ejecución (Snapshot Actual) — Table

```sql
SELECT 
	timestamp, 
	pid, 
	name, 
	ram AS "RAM (MB)", 
	cpu AS "CPU (%)"
FROM process_log 
ORDER BY timestamp DESC 
LIMIT 10;
```

8) Porcentaje de CPU en Tiempo Real (Por Contenedor) — Time Series

```sql
SELECT 
	timestamp,
	cpu,
	name || '_' || pid AS metric
FROM process_log
WHERE timestamp > datetime('now', '-2 minutes')
ORDER BY timestamp ASC;
```

Nota: En la vista 8, configurar en Grafana "Column to use as metric" → `metric`.

Tip: Con el `main.go` actualizado, el daemon intenta levantar Grafana automáticamente usando `docker-compose` (o `docker compose` si el primero falla) y ejecuta el generador de tráfico cada 60s.

## 7. Notas de Seguridad y Mantenimiento

- **Solo lectura en `/proc`:** No habilitar escritura si no hay handlers seguros.
- **`sudo` y `kill -9`:** Limitar pruebas a entornos controlados para evitar matar procesos críticos.
- **Limpieza:**
	```bash
	# Descargar los módulos
	sudo rmmod procesos || true
	sudo rmmod ram || true

	# Detener y eliminar contenedores creados por las pruebas (opcional)
	docker ps -aq --filter name=so1_contenedor_ | xargs -r docker stop
	docker ps -aq --filter name=so1_contenedor_ | xargs -r docker rm
	```

## 8. Referencias Rápidas

- `modulo-kernel/Makefile`: genera `procesos.ko` y `ram.ko` (`obj-m += procesos.o` y `obj-m += ram.o`). — [ver archivo](../modulo-kernel/Makefile)
- `go-daemon/main.go`: constantes `DESIRED_HIGH`, `DESIRED_LOW`, `PROC_FILE`, `RAM_FILE`. — [ver archivo](../go-daemon/main.go)
- `bash/generator.sh`: crea 10 contenedores a partir de `so1_ram|so1_cpu|so1_low`. — [ver archivo](../bash/generator.sh)
- `docker-files/`: `dockerfile.ram`, `dockerfile.cpu`, `dockerfile.low`. — [abrir carpeta](../docker-files/)
