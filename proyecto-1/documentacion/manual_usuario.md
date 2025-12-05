# Manual de Usuario — Proyecto 1 (SO1)

Guía rápida para construir imágenes, cargar el módulo de kernel, generar carga con contenedores y ejecutar el daemon de monitoreo.

## Índice

- [Manual de Usuario — Proyecto 1 (SO1)](#manual-de-usuario--proyecto-1-so1)
	- [Índice](#índice)
	- [Requisitos Previos](#requisitos-previos)
	- [Instalación y Ejecución](#instalación-y-ejecución)
		- [1 Construir imágenes de Docker](#1-construir-imágenes-de-docker)
		- [2 Compilar y cargar el módulo del Kernel](#2-compilar-y-cargar-el-módulo-del-kernel)
		- [3 Iniciar Monitor y Base de Datos (Go)](#3-iniciar-monitor-y-base-de-datos-go)
		- [4 Levantar Grafana](#4-levantar-grafana)
		- [5 Generar tráfico (contenedores de prueba)](#5-generar-tráfico-contenedores-de-prueba)
	- [Solución de Problemas](#solución-de-problemas)
	- [Limpieza](#limpieza)

## Requisitos Previos

- Sistema operativo: Linux (Ubuntu 22.04+ recomendado)
- Docker instalado y servicio activo
- Go (Golang) 1.20+
- Herramientas de compilación: GCC y Make (`sudo apt install build-essential`)

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

### 2 Compilar y cargar el módulo del Kernel

```bash
cd proyecto-1/modulo-kernel
make clean && make
sudo insmod procesos.ko
sudo insmod ram.ko

# Verificación
cat /proc/sysinfo_so1_202302220
cat /proc/raminfo_so1_202302220
```

Debería mostrarse un arreglo JSON con la lista de procesos y métricas.

Archivos relacionados:

- `modulo-kernel/Makefile` — [ver archivo](../modulo-kernel/Makefile)
- `modulo-kernel/procesos.c` — [ver archivo](../modulo-kernel/procesos.c)
- `modulo-kernel/ram.c` — [ver archivo](../modulo-kernel/ram.c)

### 3 Iniciar Monitor y Base de Datos (Go)

El daemon leerá los archivos ``/proc`` y guardará la info en ``metrics.db``.

```bash
cd ../go-daemon

# Instalar dependencias (solo la primera vez)
go mod init daemon
go get github.com/mattn/go-sqlite3
go mod tidy

# Ejecutar con permisos de superusuario (necesario para leer /proc protegidos)
sudo env "PATH=$PATH" go run main.go
```


### 4 Levantar Grafana

En una nueva terminal, levantamos el servicio de visualización.

```bash
cd .. 
# Crear archivo DB vacío con permisos amplios para evitar errores de lectura en Grafana
touch go-daemon/metrics.db
chmod 666 go-daemon/metrics.db

docker-compose up -d
```

Acceder a Grafana en: http://localhost:3000 (Usuario: admin / Pass: admin)

### 5 Generar tráfico (contenedores de prueba)

```bash
cd ../bash
chmod +x generator.sh
./generator.sh
```

El daemon de Go empezará a detectar los contenedores y aplicará la lógica de eliminación si se exceden los límites definidos.

Archivo relacionado: `bash/generator.sh` — [ver archivo](../bash/generator.sh)


## Solución de Problemas

- `insmod`: Invalid parameters o falla al cargar
	- Revise `dmesg | tail -n 50` para ver el motivo.
	- Asegure permisos de solo lectura (0444) en `/proc` (ya aplicado en el código).
	- Verifique headers del kernel instalados (paquete `linux-headers-$(uname -r)`).
- Docker requiere privilegios
	- Use `sudo` o agregue su usuario al grupo `docker` y reabra sesión.
- No aparece `/proc/sysinfo_so1_202302220` o `/proc/raminfo...`
	- Confirme que el módulo esté cargado: `lsmod | grep so1`
	- Intente recargar: `sudo rmmod procesos` o `sudo rmmod ram` y vuelva a hacer `insmod`
- Go no encuentra dependencias o no ejecuta
	- Verifique `go version` y `which go`. Ejecute desde `proyecto-1/go-daemon`.

## Limpieza

```bash
# 1. Detener generador y contenedores
docker stop $(docker ps -q --filter name=so1_contenedor)
docker rm $(docker ps -aq --filter name=so1_contenedor)

# 2. Bajar Grafana
docker-compose down

# 3. Descargar módulos del kernel
sudo rmmod procesos
sudo rmmod ram
```
