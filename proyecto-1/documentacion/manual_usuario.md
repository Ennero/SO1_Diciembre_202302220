# Manual de Usuario — Proyecto 1 (SO1)

Guía práctica y amigable para construir imágenes, cargar los módulos de kernel, generar carga con contenedores y ejecutar el daemon de monitoreo con visualización en Grafana.

## Índice

- [Manual de Usuario — Proyecto 1 (SO1)](#manual-de-usuario--proyecto-1-so1)
	- [Índice](#índice)
	- [Requisitos Previos](#requisitos-previos)
	- [Instalación y Ejecución](#instalación-y-ejecución)
		- [Ejecución automática (setup\_all.sh)](#ejecución-automática-setup_allsh)
		- [1 Construir imágenes de Docker](#1-construir-imágenes-de-docker)
		- [2 Compilar y cargar los módulos del Kernel](#2-compilar-y-cargar-los-módulos-del-kernel)
		- [3 Iniciar Monitor y Base de Datos (Go)](#3-iniciar-monitor-y-base-de-datos-go)
		- [4 Levantar Grafana](#4-levantar-grafana)
			- [Las 8 Consultas para Grafana (Dashboard)](#las-8-consultas-para-grafana-dashboard)
		- [5 Generar tráfico (contenedores de prueba)](#5-generar-tráfico-contenedores-de-prueba)
	- [Carpeta Compartida Host↔VM (Virtio-FS)](#carpeta-compartida-hostvm-virtio-fs)
		- [Migrar el proyecto desde carpeta compartida a Home (100% Linux)](#migrar-el-proyecto-desde-carpeta-compartida-a-home-100-linux)
	- [Solución de Problemas](#solución-de-problemas)
	- [Limpieza](#limpieza)

## Requisitos Previos

- Sistema operativo: Linux (Ubuntu 22.04+ recomendado)
- Docker instalado y servicio activo (`docker --version` para verificar)
- Go (Golang) 1.20+ (`go version` para verificar)
- Herramientas de compilación: GCC y Make (`sudo apt install build-essential`)
- Encabezados del kernel instalados: `sudo apt install linux-headers-$(uname -r)`
- Permisos para usar Docker: ejecutar con `sudo` o agregar su usuario al grupo `docker` y reiniciar sesión: `sudo usermod -aG docker $USER`

Instalar todas las dependencias (Ubuntu/Debian)

```bash
# 1. Actualizar índices
sudo apt update

# 2. Herramientas de compilación y headers (necesario para compilar ciertas dependencias)
sudo apt install -y build-essential linux-headers-$(uname -r)

# 3. Docker Engine y Docker Compose V2
# NOTA: 'docker-compose-plugin' reemplaza al antiguo paquete python 'docker-compose'
sudo apt install -y docker.io docker-compose-plugin

# (Comentado para evitar conflictos: No mezcles Snap con Apt. Usa uno u otro)
# sudo snap install docker

# 4. Habilitar y arrancar Docker
sudo systemctl enable --now docker

# 5. Configurar Docker sin sudo
# Esto agrega tu usuario actual al grupo docker
sudo usermod -aG docker "$USER"
echo "[INFO] Recuerda cerrar sesión y volver a entrar para que funcione Docker sin 'sudo'"

# 6. Instalar Go
sudo apt install -y golang

# 7. (Opcional) Crear alias para compatibilidad
# Esto permite que si escribes 'docker-compose' (viejo), ejecute 'docker compose' (nuevo)
echo 'alias docker-compose="docker compose"' >> ~/.bashrc

# 8. Verificaciones
echo "--- Verificando versiones ---"
docker --version
docker compose version   # Nota: El comando nuevo es SIN guion
go version
```

Iniciar el servicio de Docker (si no está activo)

```bash
# Arrancar el servicio Docker
sudo systemctl start docker

# Verificar estado (debe indicar "Active: active (running)")
sudo systemctl status docker
```

## Instalación y Ejecución

### Ejecución automática (setup_all.sh)

Si prefieres automatizar todo (montaje de carpeta compartida, migración a Home, instalación de dependencias, build de imágenes, compilación/carga de módulos, Grafana y daemon), usa el script:

```bash
# Desde la raíz del repositorio
cd proyecto-1/bash
chmod +x setup_all.sh
./setup_all.sh
```

Notas:
- El script intenta montar `virtiofs` con el Target `micarpeta` y, si falla, usa `9p`. Puedes editar `proyecto-1/bash/setup_all.sh` para ajustar `SHARED_TARGET_NAME` y `SHARED_MOUNTPOINT`.
- Si es la primera vez que se agrega tu usuario al grupo `docker`, cierra sesión y vuelve a entrar para que aplique.
- El daemon se ejecuta al final en primer plano. Para detenerlo, usa `Ctrl+C`.

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
cat /proc/continfo_so1_202302220
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
sudo docker-compose up -d
```

Acceder a Grafana en: http://localhost:3000 (Usuario: `admin` / Password: `admin`)

Nota: el `docker-compose.yml` monta la base desde `../go-daemon/metrics.db` hacia `/var/lib/grafana/metrics.db` en modo lectura. Asegúrate de crear el archivo en `proyecto-1/go-daemon/metrics.db` antes de `docker-compose up -d`.

Configuración rápida en Grafana:
- Añade un Data Source de tipo "SQLite" (el plugin ya se instala automáticamente).
- Ruta del archivo: `/var/lib/grafana/metrics.db` (montado en el contenedor).
- Guarda y prueba. Ejemplos de consultas:

#### Las 8 Consultas para Grafana (Dashboard)

1) Monitor de RAM en Tiempo Real (Dientes de Sierra)
- Visualización: Time Series (líneas)
- Qué muestra: Evolución del porcentaje de RAM del sistema.

```sql
SELECT 
	timestamp,
	percentage AS "Uso RAM %"
FROM ram_log 
ORDER BY timestamp ASC;
```

2) Consumo de RAM en MB (Actual)
- Visualización: Gauge o Stat
- Qué muestra: MB usados actualmente.

```sql
SELECT 
	used AS "MB Usados"
FROM ram_log 
ORDER BY timestamp DESC 
LIMIT 1;
```

3) Total de Contenedores Eliminados
- Visualización: Stat
- Qué muestra: Conteo histórico de eliminaciones (Thanos).

```sql
SELECT COUNT(*) AS "Total Eliminados" FROM kill_log;
```

4) Historial de Eliminaciones (Log de Muertes)
- Visualización: Table
- Qué muestra: Quién se mató, cuándo y por qué.

```sql
SELECT 
	timestamp AS "Fecha/Hora", 
	pid AS "PID", 
	name AS "Nombre Contenedor", 
	reason AS "Razón"
FROM kill_log 
ORDER BY timestamp DESC;
```

5) Top Contenedores por Consumo de CPU (Histórico)
- Visualización: Bar Gauge
- Qué muestra: Máximo %CPU registrado por contenedor.

```sql
SELECT 
	name || ' (' || pid || ')' AS Container, 
	MAX(cpu) AS "Max CPU %"
FROM process_log 
GROUP BY pid, name
ORDER BY "Max CPU %" DESC 
LIMIT 5;
```

6) Top Contenedores por Consumo de RAM (Histórico)
- Visualización: Bar Gauge
- Qué muestra: Máximo RAM MB por contenedor.

```sql
SELECT 
	name || ' (' || pid || ')' AS Container, 
	MAX(ram) AS "Max RAM MB"
FROM process_log 
GROUP BY pid, name
ORDER BY "Max RAM MB" DESC 
LIMIT 5;
```

7) Tabla de Procesos en Ejecución (Snapshot Actual)
- Visualización: Table
- Qué muestra: Últimos procesos registrados con RAM y CPU.

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

8) Porcentaje de CPU en Tiempo Real (Por Contenedor)
- Visualización: Time Series
- Qué muestra: Serie temporal por contenedor (métrica por nombre+pid).

```sql
SELECT 
	timestamp,
	cpu,
	name || '_' || pid AS metric
FROM process_log
WHERE timestamp > datetime('now', '-2 minutes')
ORDER BY timestamp ASC;
```

Nota: En la visualización 8, configura en Grafana la opción "Column to use as metric" → `metric`.

Tip: Si usas el daemon modificado (ver `go-daemon/main.go`), Grafana se levanta automáticamente y el generador de tráfico corre cada 60s.

### 5 Generar tráfico (contenedores de prueba)

```bash
cd ../bash
chmod +x generator.sh
./generator.sh
```

El daemon de Go detectará los contenedores creados y aplicará la lógica de eliminación si se exceden los límites definidos.

Archivo relacionado: `bash/generator.sh` — [ver archivo](../bash/generator.sh)


## Carpeta Compartida Host↔VM (Virtio-FS)

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

### Migrar el proyecto desde carpeta compartida a Home (100% Linux)

Para evitar problemas de permisos y rendimiento, se recomienda copiar el proyecto desde la carpeta compartida a tu `Home` dentro de la VM y trabajar desde ahí:

```bash
# 1. Ir a tu carpeta personal (Home)
cd ~

# 2. Copiar todo el proyecto desde la carpeta compartida hacia aquí
cp -r /mnt/compartido/proyecto-1 .

# 3. Entrar a la nueva copia (que ya es 100% Linux)
cd proyecto-1

```


## Solución de Problemas

- `insmod`: parámetros inválidos o falla al cargar
	- Revise `dmesg | tail -n 50` para ver el motivo.
	- Verifique que los encabezados del kernel estén instalados (`linux-headers-$(uname -r)`).
	- Si los nodos `/proc` no tienen permisos correctos, recompilar; el código ya define `0444` (solo lectura).
- Docker requiere privilegios
	- Use `sudo` o agregue su usuario al grupo `docker` y reabra sesión.
- No aparece `/proc/sysinfo_so1_202302220` o `/proc/continfo_so1_202302220`
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
