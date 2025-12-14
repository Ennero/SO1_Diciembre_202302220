# Sistemas Operativos 1 - Diciembre 2025
**Carnet:** 202302220

Repositorio de proyectos prácticos para el curso de Sistemas Operativos 1.

---

## 📂 Proyecto 1: Monitor de Contenedores y Kernel

Sistema integral de monitoreo que cruza información de **Espacio de Kernel (C)** y **Espacio de Usuario (Go)** para gestionar el ciclo de vida de contenedores Docker basándose en consumo de recursos.

### 📋 Características
- **Módulos de Kernel:** Lectura directa de `task_struct` y `sysinfo` vía `/proc`.
- **Daemon en Go:** Orquestador que carga módulos, gestiona Cronjobs y conecta con Docker.
- **Visualización:** Dashboards en **Grafana** (Contenedores y Sistema) alimentados por SQLite.
- **Automatización:** Generación de tráfico de contenedores (High/Low) automática.

### 📄 Documentación
Para detalles profundos y guías paso a paso, consulte:
- [📖 Manual de Usuario](proyecto-1/documentacion/manual_usuario.md)
- [🛠 Manual Técnico](proyecto-1/documentacion/manual_tecnico.md)

### 🚀 Inicio Rápido

**1. Construcción de Imágenes y Módulos**
El sistema necesita 3 imágenes base para generar tráfico. Ejecute desde la carpeta raíz del proyecto:
```bash
cd proyecto-1
docker build -t so1_ram -f bash/docker-files/dockerfile.ram .
docker build -t so1_cpu -f bash/docker-files/dockerfile.cpu .
docker build -t so1_low -f bash/docker-files/dockerfile.low .
```

**Verificación**: Ejecute `docker images | grep so1_` para confirmar.

### Paso 2: Compilar Módulos del Kernel

Antes de iniciar el daemon, debemos compilar los archivos `.c` a objetos de kernel `.ko`.
```bash
cd modulo-kernel
make clean && make
```

Esto generará `sysinfo.ko` y `continfo.ko`.

> **Nota**: No es necesario cargarlos manualmente (`insmod`), el Daemon de Go lo hará automáticamente.

### Paso 3: Iniciar el Daemon (Go)

El daemon es el cerebro del proyecto: carga los módulos, configura el cronjob, levanta Grafana y monitorea el sistema.
```bash
cd ../go-daemon
go mod tidy

# Ejecutar con SUDO (Necesario para insmod y acceso a /proc)
sudo go run main.go
```
