# Manual de Usuario — Proyecto 1 (SO1)

Guía rápida para construir imágenes, cargar el módulo de kernel, generar carga con contenedores y ejecutar el daemon de monitoreo.

## Requisitos Previos

- Sistema operativo: Linux (Ubuntu 22.04+ recomendado)
- Docker instalado y servicio activo
- Go (Golang) 1.20+
- Herramientas de compilación: GCC y Make

## Instalación y Ejecución

### 1 Construir imágenes de Docker

Desde la raíz del repositorio o entrando a `proyecto-1`:

```bash
cd proyecto-1
docker build -t so1_ram -f docker-files/dockerfile.ram .
docker build -t so1_cpu -f docker-files/dockerfile.cpu .
docker build -t so1_low -f docker-files/dockerfile.low .
```

### 2 Compilar y cargar el módulo del Kernel

```bash
cd proyecto-1/modulo-kernel
make clean && make
sudo insmod module.ko

# Verificación
cat /proc/continfo_so1_202302220
```

Debería mostrarse un arreglo JSON con la lista de procesos y métricas.

### 3 Generar tráfico (contenedores de prueba)

```bash
cd ../bash
chmod +x generator.sh
./generator.sh
```

### 4 Iniciar el monitor (Daemon en Go)

```bash
cd ../go-daemon
sudo env "PATH=$PATH" go run main.go
```

Resultado esperado: Actualizaciones cada 5 segundos con contenedores detectados, consumo de RAM, %CPU, y mensajes si se eliminan procesos/containers sobrantes.

## Solución de Problemas

- `insmod`: Invalid parameters o falla al cargar
	- Revise `dmesg | tail -n 50` para ver el motivo.
	- Asegure permisos de solo lectura (0444) en `/proc` (ya aplicado en el módulo).
	- Verifique headers del kernel instalados (paquete `linux-headers-$(uname -r)`).
- Docker requiere privilegios
	- Use `sudo` o agregue su usuario al grupo `docker` y reabra sesión.
- No aparece `/proc/continfo_so1_202302220`
	- Confirme que el módulo esté cargado: `lsmod | grep module` y `sudo rmmod module` para recargar si es necesario.
- Go no encuentra dependencias o no ejecuta
	- Verifique `go version` y `which go`. Ejecute desde `proyecto-1/go-daemon`.

## Limpieza

```bash
# Descargar módulo
sudo rmmod module

# Detener y eliminar contenedores de prueba (opcionales)
docker ps -aq --filter name=so1_contenedor_ | xargs -r docker stop
docker ps -aq --filter name=so1_contenedor_ | xargs -r docker rm
```

## Próximo Paso (opcional)

- Persistir métricas en SQLite desde el daemon y exponerlas en Grafana con dashboards. ¿Quieres que agregue la persistencia en Go ahora?