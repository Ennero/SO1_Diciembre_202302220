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

## 📦 Proyecto 2: Sistema Distribuido de Ventas (Kubernetes)

Arquitectura de microservicios para ingesta, procesamiento y visualización de ventas a alta concurrencia. Incluye API en Rust, clientes/servidores en Go, Kafka, consumidor en Go, Valkey en VM con KubeVirt, dashboards en Grafana y generación de carga con Locust.

### 🧩 Componentes
- **API Rust (HTTP):** [proyecto-2/api-rust](proyecto-2/api-rust)
- **Cliente HTTP→gRPC (Go):** [proyecto-2/go-http-client](proyecto-2/go-http-client)
- **Servidor gRPC (Go):** [proyecto-2/grpc-server-go](proyecto-2/grpc-server-go)
- **Consumidor Kafka (Go):** [proyecto-2/kafka-consumer-go](proyecto-2/kafka-consumer-go)
- **Kafka (Strimzi):** manifiestos en [proyecto-2/k8s](proyecto-2/k8s)
- **Valkey (VM KubeVirt):** [proyecto-2/k8s/valkey-vm.yaml](proyecto-2/k8s/valkey-vm.yaml)
- **Grafana:** guía en [proyecto-2/grafana/grafana.md](proyecto-2/grafana/grafana.md) y dashboard en [proyecto-2/grafana/grafico_ventas.json](proyecto-2/grafana/grafico_ventas.json)
- **Locust:** [proyecto-2/locust/locustfile.py](proyecto-2/locust/locustfile.py)
- **Protobuf:** [proyecto-2/proto/ventas.proto](proyecto-2/proto/ventas.proto)

### 📄 Documentación
- [🛠 Manual Técnico P2](proyecto-2/docs/manual-tecnico.md)
- [📘 README Manifiestos K8s](proyecto-2/k8s/README.md)

### 🚀 Despliegue Rápido (resumen)
Requiere un registry accesible por el clúster y operadores de Strimzi y KubeVirt instalados. Ajuste las referencias `image:` en los YAML si usa un registry privado.

```bash
# Namespace e infraestructura base
kubectl create namespace blackfriday
kubectl apply -f proyecto-2/k8s/strimzi-kafka.yaml
kubectl apply -f proyecto-2/k8s/kafka-topic.yaml

# Base de datos en VM (KubeVirt)
kubectl apply -f proyecto-2/k8s/valkey-vm.yaml

# Aplicaciones y red
kubectl apply -f proyecto-2/k8s/apps.yaml
kubectl apply -f proyecto-2/k8s/go-client.yaml
kubectl apply -f proyecto-2/k8s/consumer.yaml
kubectl apply -f proyecto-2/k8s/grafana.yaml
kubectl apply -f proyecto-2/k8s/ingress.yaml
kubectl apply -f proyecto-2/k8s/hpa.yaml

# Verificación
kubectl get pods -n blackfriday
kubectl get vmi -n blackfriday
kubectl get svc,ingress -n blackfriday
```

### 🧪 Pruebas de carga (Locust)
1) Obtenga la IP pública del Ingress de NGINX.
```bash
kubectl get svc -n ingress-nginx
```
2) Ejecute Locust desde su máquina local:
```bash
cd proyecto-2/locust
locust -f locustfile.py
```
3) Abra http://localhost:8089 y configure Host con la IP externa del Ingress. Inicie con 200 usuarios y `spawn rate` 10.

### 📊 Grafana
- Despliegue y service: [proyecto-2/k8s/grafana.yaml](proyecto-2/k8s/grafana.yaml). Dashboard de referencia: [proyecto-2/grafana/grafico_ventas.json](proyecto-2/grafana/grafico_ventas.json) y guía: [proyecto-2/grafana/grafana.md](proyecto-2/grafana/grafana.md).
