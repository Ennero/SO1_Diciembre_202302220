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
	- [7. Notas de Seguridad y Mantenimiento](#7-notas-de-seguridad-y-mantenimiento)
	- [8. Referencias Rápidas](#8-referencias-rápidas)

## 1. Arquitectura del Sistema

- **Espacio de Kernel (C):** Módulo que recorre procesos (`task_struct`) y expone métricas en un archivo virtual en `/proc` en formato JSON.
- **Espacio de Usuario (Go):** Daemon que lee el JSON del kernel, obtiene contenedores desde Docker, calcula %CPU por diferencia y aplica política de control (matar excedentes).

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

$$
\%\,CPU = \frac{\Delta(utime + stime)}{\Delta t \times HZ} \times 100
$$

- `Δ(utime + stime)`: Diferencia de ticks de CPU entre lecturas.
- `Δt`: Tiempo real entre lecturas (segundos).
- `HZ`: Ticks por segundo del sistema (en la práctica del daemon se usa `HZ = 100`).

Nota: `HZ` puede variar según la distro (p. ej., 100, 250, 1000). Para mayor precisión, puede leerse en tiempo de ejecución usando `getconf CLK_TCK` y ajustar el cálculo en el daemon.

### 3.2. Política de control 

- **Constantes:** `DESIRED_HIGH = 2`, `DESIRED_LOW = 3`.
- **Acción:** Si hay más procesos de los deseados en cada categoría, se eliminan los sobrantes con `kill -9 <PID>`.

## 4. Automatización (Bash)

- **Ubicación:** `proyecto-1/bash/generator.sh` — [ver archivo](../bash/generator.sh)
- **Función:** Estresar el sistema para pruebas creando 10 contenedores aleatorios basados en las imágenes `so1_ram`, `so1_cpu`, `so1_low`. Nombres únicos para facilitar rastreo.

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
sudo docker build -t so1_ram -f docker-files/dockerfile.ram .
sudo docker build -t so1_cpu -f docker-files/dockerfile.cpu .
sudo docker build -t so1_low -f docker-files/dockerfile.low .
```

Archivos Dockerfiles:

- `docker-files/dockerfile.ram` — [ver archivo](../docker-files/dockerfile.ram)
- `docker-files/dockerfile.cpu` — [ver archivo](../docker-files/dockerfile.cpu)
- `docker-files/dockerfile.low` — [ver archivo](../docker-files/dockerfile.low)

### 6.3. Compilación y Carga de los Módulos

```bash
cd proyecto-1/modulo-kernel
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

**Resultado esperado:** Cada 5 segundos se listan contenedores detectados, RAM, `%CPU` calculado y, en caso de exceso, mensajes de eliminación de procesos sobrantes.

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
