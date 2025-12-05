# Manual de Usuario — Proyecto 1 (SO1)

Guía práctica y amigable para construir imágenes, cargar los módulos de kernel, generar carga con contenedores y ejecutar el daemon de monitoreo con visualización en Grafana.

## Índice

- [Manual de Usuario — Proyecto 1 (SO1)](#manual-de-usuario--proyecto-1-so1)
	- [Índice](#índice)
	- [Requisitos Previos](#requisitos-previos)
	- [Instalación y Ejecución](#instalación-y-ejecución)
		- [1 Construir imágenes de Docker](#1-construir-imágenes-de-docker)
		- [2 Compilar y cargar los módulos del Kernel](#2-compilar-y-cargar-los-módulos-del-kernel)
		- [3 Iniciar Monitor y Base de Datos (Go)](#3-iniciar-monitor-y-base-de-datos-go)
		- [4 Levantar Grafana](#4-levantar-grafana)
		- [5 Generar tráfico (contenedores de prueba)](#5-generar-tráfico-contenedores-de-prueba)
	- [Solución de Problemas](#solución-de-problemas)
	- [Limpieza](#limpieza)

## Requisitos Previos

- Sistema operativo: Linux (Ubuntu 22.04+ recomendado)
- Docker instalado y servicio activo (`docker --version` para verificar)
- Go (Golang) 1.20+ (`go version` para verificar)
- Herramientas de compilación: GCC y Make (`sudo apt install build-essential`)
- Encabezados del kernel instalados: `sudo apt install linux-headers-$(uname -r)`
- Permisos para usar Docker: ejecutar con `sudo` o agregar su usuario al grupo `docker` y reiniciar sesión: `sudo usermod -aG docker $USER`

## Instalación y Ejecución

### 1 Construir imágenes de Docker

Desde la raíz del repositorio o entrando a `proyecto-1`:

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

### 2 Compilar y cargar los módulos del Kernel

```bash
cd proyecto-1/modulo-kernel
make clean && make

# Cargar módulos (requiere privilegios)
sudo insmod procesos.ko
sudo insmod ram.ko

# Verificación de dispositivos /proc
cat /proc/sysinfo_so1_202302220
cat /proc/raminfo_so1_202302220
```

Debería mostrarse un arreglo JSON con la lista de procesos y métricas. Si hay errores al cargar, revise `dmesg | tail -n 50`.

Archivos relacionados:

- `modulo-kernel/Makefile` — [ver archivo](../modulo-kernel/Makefile)
- `modulo-kernel/procesos.c` — [ver archivo](../modulo-kernel/procesos.c)
- `modulo-kernel/ram.c` — [ver archivo](../modulo-kernel/ram.c)

### 3 Iniciar Monitor y Base de Datos (Go)

El daemon lee los archivos de `/proc` y guarda la información en `metrics.db` (SQLite).

```bash
cd proyecto-1/go-daemon

# Instalar dependencias (solo la primera vez)
# Nota: si ya existe `go.mod`, no ejecute `go mod init`
go get github.com/mattn/go-sqlite3
go mod tidy

# Ejecutar con permisos de superusuario (lectura de /proc y manejo de contenedores)
sudo env "PATH=$PATH" go run main.go
```


### 4 Levantar Grafana

En una nueva terminal, levantamos el servicio de visualización.

```bash
cd proyecto-1

# Crear archivo DB vacío con permisos amplios para evitar errores de lectura en Grafana
touch go-daemon/metrics.db
chmod 666 go-daemon/metrics.db

# Levantar stack de Grafana desde el directorio correspondiente
cd dashboard
docker-compose up -d
```

Acceder a Grafana en: http://localhost:3000 (Usuario: `admin` / Password: `admin`)

Nota: el `docker-compose.yml` monta la base desde `../go-daemon/metrics.db` hacia `/var/lib/grafana/metrics.db` en modo lectura. Asegúrate de crear el archivo en `proyecto-1/go-daemon/metrics.db` antes de `docker-compose up -d`.

Configuración rápida en Grafana:
- Añade un Data Source de tipo "SQLite" (el plugin ya se instala automáticamente).
- Ruta del archivo: `/var/lib/grafana/metrics.db` (montado en el contenedor).
- Guarda y prueba. Ejemplos de consultas:
	- `SELECT timestamp, percentage FROM ram_log ORDER BY timestamp DESC LIMIT 50;`
	- `SELECT timestamp, name, ram, cpu FROM process_log ORDER BY timestamp DESC LIMIT 50;`

### 5 Generar tráfico (contenedores de prueba)

```bash
cd ../bash
chmod +x generator.sh
./generator.sh
```

El daemon de Go detectará los contenedores creados y aplicará la lógica de eliminación si se exceden los límites definidos.

Archivo relacionado: `bash/generator.sh` — [ver archivo](../bash/generator.sh)


## Solución de Problemas

- `insmod`: parámetros inválidos o falla al cargar
	- Revise `dmesg | tail -n 50` para ver el motivo.
	- Verifique que los encabezados del kernel estén instalados (`linux-headers-$(uname -r)`).
	- Si los nodos `/proc` no tienen permisos correctos, recompilar; el código ya define `0444` (solo lectura).
- Docker requiere privilegios
	- Use `sudo` o agregue su usuario al grupo `docker` y reabra sesión.
- No aparece `/proc/sysinfo_so1_202302220` o `/proc/raminfo_so1_202302220`
	- Confirme que los módulos estén cargados: `lsmod | grep so1`
	- Intente recargar: `sudo rmmod procesos` y/o `sudo rmmod ram` y vuelva a hacer `insmod`.
- Go no encuentra dependencias o no ejecuta
	- Verifique `go version` y `which go`. Ejecute los comandos dentro de `proyecto-1/go-daemon`.
- Grafana no muestra datos
	- Asegure que `go-daemon/metrics.db` existe y tiene permisos `666`.
	- Verifique que el daemon está corriendo y escribiendo en la base (`sudo env "PATH=$PATH" go run main.go`).

## Limpieza

```bash
# 1. Detener generador y contenedores de prueba
docker stop $(docker ps -q --filter name=so1_contenedor) || true
docker rm $(docker ps -aq --filter name=so1_contenedor) || true

# 2. Bajar Grafana (desde `proyecto-1/dashboard`)
cd proyecto-1/dashboard
docker-compose down

# 3. Descargar módulos del kernel
sudo rmmod procesos || true
sudo rmmod ram || true
```

¡Listo! Con estos pasos, el entorno queda limpio y preparado para una nueva ejecución.
