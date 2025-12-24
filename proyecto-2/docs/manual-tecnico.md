# Manual Técnico - Proyecto 2: Sistema Distribuido de Procesamiento de Ventas en Kubernetes
## Índice

1. [Introducción](#sección-1-introducción)
2. [Arquitectura del Sistema](#sección-2-arquitectura-del-sistema)
3. [Documentación de Deployments](#sección-3-documentación-de-deployments)
4. [Descripción Detallada de la Implementación y Despliegue](#sección-4-descripción-detallada-de-la-implementación-y-despliegue)
5. [Instrucciones de Despliegue (Resumen)](#sección-5-instrucciones-de-despliegue-resumen)
6. [Desarrollo y Retos Encontrados](#sección-6-desarrollo-y-retos-encontrados)
7. [Análisis de Rendimiento y Réplicas](#sección-7-análisis-de-rendimiento-y-réplicas)
8. [Conclusiones y Comparativas](#sección-8-conclusiones-y-comparativas)

## SECCIÓN 1: INTRODUCCIÓN

El presente proyecto consiste en la implementación de una arquitectura de microservicios distribuida, desplegada sobre Google Kubernetes Engine (GKE). El sistema simula un entorno de alto tráfico (ventas de Black Friday) utilizando Locust para generar carga, procesando las transacciones a través de APIs en Rust y Go, desacoplando la comunicación mediante Kafka, y persistiendo los datos en una base de datos en memoria (Valkey) ejecutada sobre una máquina virtual real gestionada por KubeVirt. Finalmente, la visualización de métricas en tiempo real se realiza mediante Grafana.

## SECCIÓN 2: ARQUITECTURA DEL SISTEMA

La solución implementada sigue un patrón de microservicios con comunicación híbrida (Síncrona/Asíncrona).

### Componentes Principales:

- **Generador de Carga (Locust)**: Simula miles de clientes enviando transacciones JSON vía HTTP.

- **Ingress Controller (NGINX) & API Gateway**: El punto de entrada al clúster se gestiona mediante un NGINX Ingress Controller. Este componente recibe el tráfico externo HTTP en la IP pública y lo enruta internamente hacia el servicio de la API de Rust.
  - **Ingress Resource:** Define las reglas de ruteo.
  - **API Rust:** Se expone ahora como ClusterIP (solo accesible desde dentro del clúster), mejorando la seguridad al no exponer el pod directamente a internet.

- **Backend de Procesamiento (Go)**: Servidor gRPC que recibe datos de Rust y actúa como Productor para Kafka.

- **Message Broker (Strimzi Kafka)**: Clúster de Kafka que gestiona la cola de mensajes (Tópico: ventas), garantizando que no se pierdan datos ante picos de tráfico.

- **Consumidor (Go)**: Servicio que lee de Kafka, realiza cálculos matemáticos (promedios, tops) y escribe en la base de datos.

- **Base de Datos (Valkey sobre KubeVirt)**: Instancia de Valkey corriendo dentro de una Máquina Virtual (VM) real orquestada por Kubernetes, utilizando una imagen de disco personalizada (Alpine Linux) para persistencia.

- **Visualización (Grafana)**: Dashboard conectado a Valkey para monitoreo en tiempo real.




## SECCIÓN 3: DOCUMENTACIÓN DE DEPLOYMENTS

A continuación se describen los manifiestos de Kubernetes utilizados, ubicados en la carpeta k8s:

### namespace.yaml (blackfriday)

**Descripción**: Crea un espacio de trabajo aislado para todos los componentes del proyecto, facilitando la gestión y limpieza de recursos.

### apps.yaml (Rust y Go Server)

**Descripción**: Define los Deployments para la API de Rust y el Servidor gRPC de Go.

**Configuración Clave**:
- Rust: Se conecta a Go mediante la variable grpc-go-service:50051.
- Go: Se conecta a Kafka mediante la variable KAFKA_BROKER.
- Escalabilidad: Configurados con límites de recursos (CPU/Memory) para permitir escalado horizontal.

### consumer.yaml

**Descripción**: Despliega el consumidor de Go encargado de la lógica de negocio.

**Detalle Técnico**: Utiliza la versión v4 de la imagen, la cual incluye la lógica para calcular promedios y separar estadísticas por terminación de carnet (Electronica).

### valkey-vm.yaml (KubeVirt)

**Descripción**: Define el objeto VirtualMachine. A diferencia de un Pod normal, esto levanta una VM completa.

- **Almacenamiento**: Utiliza un containerDisk personalizado (valkey-custom-disk) basado en Alpine Linux, donde se pre-instaló y configuró Valkey.
- **Red**: Expone el servicio en el puerto 6379 para acceso interno del clúster.

### strimzi-kafka.yaml

**Descripción**: Utiliza el operador Strimzi para desplegar un clúster de Kafka en modo KRaft (sin ZooKeeper), optimizando recursos.

### hpa.yaml (Horizontal Pod Autoscaler)

**Descripción:** Configura el escalado automático para los servicios de Rust y Go. Configuración:

**Métrica:** Uso de CPU.

**Umbral:** 50%.

**Escalado:** Mínimo 1 réplica, Máximo 5 réplicas. Esto garantiza que, durante los picos de carga generados por Locust, Kubernetes aprovisione automáticamente más pods para manejar la demanda y los elimine cuando la carga baje.




## SECCIÓN 4: DESCRIPCIÓN DETALLADA DE LA IMPLEMENTACIÓN Y DESPLIEGUE

El ciclo de vida del despliegue se divide en cinco fases críticas: Preparación de Virtualización, Construcción de Artefactos (Build), Publicación (Push), Aprovisionamiento de Infraestructura (Deploy) y Ejecución de Pruebas (Run). A continuación se detalla el procedimiento técnico efectuado en cada fase.

### FASE 1: PREPARACIÓN DE LA IMAGEN DE DISCO (VIRTUALIZACIÓN)

Antes de trabajar con Kubernetes, fue necesario crear manualmente el disco virtual para Valkey.

#### Creación del entorno base

Se descargó una imagen ISO de Alpine Linux y se arrancó una máquina virtual local con QEMU para preparar el sistema operativo invitado.

```bash
qemu-system-x86_64 -cdrom alpine-virt-3.20.0-x86_64.iso -hda valkey_disk.qcow2 -m 512
```

#### Configuración interna (Dentro de la VM Alpine)

Una vez dentro de Alpine, se ejecutaron los siguientes comandos para instalar la base de datos y configurar la red para permitir acceso externo:

```bash
apk update
apk add valkey
sed -i 's/^bind 127.0.0.1 -::1/bind 0.0.0.0/' /etc/valkey.conf
sed -i 's/^protected-mode yes/protected-mode no/' /etc/valkey.conf
poweroff
```

#### Empaquetado del Disco (ContainerDisk)

Para que Kubernetes pueda gestionar este archivo de disco, se encapsuló dentro de una imagen de Docker. Se creó un Dockerfile especial (FROM scratch) que simplemente copia el archivo .qcow2 al directorio /disk/.

Con el archivo valkey_disk.qcow2 listo, nos movimos a la carpeta create-valkey-image y construimos la imagen Docker especial que transportará este disco.

```bash
cd create-valkey-image
docker build -t <IP-EXTERNA-DE-TU-VM>:5000/valkey-custom-disk:v2-valkey .
docker push <IP-EXTERNA-DE-TU-VM>:5000/valkey-custom-disk:v2-valkey
```

### FASE 2: CONSTRUCCIÓN Y EMPAQUETADO (DOCKER)

Para garantizar la portabilidad del código escrito en Rust y Go, se utilizó una estrategia de "Multi-Stage Build" en Docker.

#### Compilación (Stage Builder)

Se utilizaron imágenes base completas (como golang:1.23 o rust:1.83) que contienen todos los compiladores y herramientas de desarrollo. En esta etapa se copian los archivos fuente y se compilan los binarios estáticos.

- **En Rust**: Se compilaron las definiciones Protobuf (.proto) usando tonic-build y luego el binario del servidor.
- **En Go**: Se deshabilitó CGO (CGO_ENABLED=0) para crear un binario puramente estático que no dependa de librerías del sistema operativo host.

#### Empaquetado Final (Stage Runner)

Se extrajeron únicamente los binarios compilados y se colocaron en imágenes "Distroless" o "Slim" (basadas en Debian 12 minimalista). Esto reduce el tamaño de la imagen final (de >1GB a <100MB) y mejora la seguridad al no incluir shells ni herramientas innecesarias.

#### API Gateway (Rust)

Se compila el servidor web Actix y el cliente gRPC.

```bash
cd ../api-rust
docker build -t <IP-EXTERNA-DE-TU-VM>:5000/api-rust:v2 .
docker push <IP-EXTERNA-DE-TU-VM>:5000/api-rust:v2
```

#### Servidor Backend (Go gRPC)

Se compila el servidor que recibe peticiones y escribe en Kafka.

```bash
cd ../grpc-server-go
docker build -t <IP-EXTERNA-DE-TU-VM>:5000/grpc-server-go:v2 .
docker push <IP-EXTERNA-DE-TU-VM>:5000/grpc-server-go:v2
```

#### Consumidor (Go Kafka Consumer)

Se compila la última versión (v4) que incluye la lógica de promedios matemáticos y estadísticas por carnet.

```bash
cd ../kafka-consumer-go
docker build -t <IP-EXTERNA-DE-TU-VM>:5000/kafka-consumer-go:v4 .
docker push <IP-EXTERNA-DE-TU-VM>:5000/kafka-consumer-go:v4
```

### FASE 3: PUBLICACIÓN EN REGISTRY PRIVADO (ZOT)

Kubernetes necesita descargar las imágenes desde un servidor centralizado. Se configuró un Container Registry (Zot) en una Máquina Virtual externa con la IP <IP-EXTERNA-DE-TU-VM>.

#### Procedimiento de Subida

Se etiquetaron todas las imágenes locales apuntando a esta IP y se subieron mediante el comando docker push.

Ejemplo:
```bash
docker push <IP-EXTERNA-DE-TU-VM>:5000/api-rust:v2
```

Esto hace que los binarios compilados y el disco virtual estén disponibles vía red para el clúster de GKE.

### FASE 4: ORQUESTACIÓN EN KUBERNETES (GKE)

El despliegue se realiza aplicando los manifiestos YAML. El flujo interno que sigue Kubernetes es el siguiente:

#### Ubicación

Nos posicionamos en la carpeta de manifiestos.

```bash
cd ../k8s
```

#### Creación del Namespace

El archivo apps.yaml define primero el Namespace "blackfriday", creando un entorno aislado lógicamente.

#### Despliegue de Infraestructura Base (Kafka y Namespace)

Primero se aplica el namespace (definido dentro de apps.yaml) y el clúster de Kafka (Strimzi).

```bash
kubectl apply -f strimzi-kafka.yaml
kubectl apply -f kafka-topic.yaml
```

Verificación del estado de Kafka:

```bash
kubectl get pods -n blackfriday
```

(Esperamos a que el pod 'my-cluster-...' esté en estado Running).

#### Descarga de Imágenes (Image Pull)

Los nodos del clúster (Workers) leen la especificación de los Deployments. Al encontrar la instrucción image: <IP-EXTERNA-DE-TU-VM>:5000/..., el motor de contenedor del nodo conecta con nuestro Registry privado, descarga las capas de la imagen y arranca los contenedores.

#### Despliegue de la VM (KubeVirt)

Al aplicar valkey-vm.yaml, el operador de KubeVirt detecta el objeto VirtualMachine.

```bash
kubectl apply -f valkey-vm.yaml
```

El proceso interno es:
1. Descarga la imagen valkey-custom-disk desde el registry.
2. Extrae el archivo QCOW2.
3. Lanza un proceso qemu-kvm dentro de un Pod especial, adjuntando ese disco como si fuera un disco duro físico.
4. Finalmente, el Servicio valkey-service enruta el tráfico TCP del puerto 6379 hacia esta VM.

Verificación de la VM:

```bash
kubectl get vmi -n blackfriday
```

(Debe mostrar el estado 'Running' y tener una IP asignada).

#### Despliegue de Aplicaciones y Monitoreo

Finalmente, levantamos la API, el Servidor gRPC, el Consumidor y Grafana.

```bash
kubectl apply -f apps.yaml
kubectl apply -f consumer.yaml
kubectl apply -f grafana.yaml
```

Verificación Final de Todos los Pods:

```bash
kubectl get pods -n blackfriday
```

(Todos los componentes deben estar en estado 1/1 Running).

#### Interconexión de Servicios

- La API Rust descubre al servidor Go usando el nombre DNS interno grpc-go-service.
- El servidor Go descubre a Kafka usando el nombre DNS my-cluster-kafka-bootstrap.
- El Consumidor descubre a Valkey usando el nombre DNS valkey-service.

### FASE 5: EJECUCIÓN DE PRUEBAS DE CARGA (LOCUST)

Para simular el tráfico de Black Friday, se ejecuta Locust desde la máquina local del desarrollador.

#### Obtención de la IP Pública (Vía Ingress)

Como el sistema está protegido por un Ingress Controller, debemos obtener la IP externa del balanceador de carga de NGINX, no del servicio interno.

```bash
kubectl get svc -n ingress-nginx
```

(Copiamos la EXTERNAL-IP de 'api-rust-service', ejemplo: <IP-EXTERNA-DE-TU-VM>).

#### Ejecución del Script de Ataque

```bash
cd ../locust
locust -f locustfile.py
```

#### Interfaz Web

Abrimos un navegador en http://localhost:8089 configuramos:
- Host: http://<IP-EXTERNA-DE-TU-VM>
- Usuarios: 200
- Spawn Rate: 10

Y presionamos "Start Swarming".

#### Flujo de Datos en Tiempo Real

El flujo de datos en tiempo real funciona así:

1. Locust (Cliente) genera miles de peticiones HTTP POST hacia la IP pública del LoadBalancer de Rust.
2. Rust recibe el JSON, lo deserializa y lo envía síncronamente vía gRPC a Go.
3. Go serializa el mensaje a bytes y lo inyecta asíncronamente en el tópico "ventas" de Kafka.
4. El Consumidor (Go) monitorea el tópico. Al llegar un mensaje, lo procesa, actualiza los contadores y listas ordenadas (Sorted Sets) en Valkey.
5. Grafana consulta periódicamente a Valkey y actualiza los gráficos visuales.

#### Monitoreo de Logs en Tiempo Real

Para verificar que el sistema procesa los datos, observamos los logs del consumidor en el clúster:

```bash
kubectl logs -n blackfriday -l app=kafka-consumer -f
```

(Salida esperada: "Procesado: Mouse-Gamer (Cant: 2, $50.00) - Electronica").

## SECCIÓN 5: INSTRUCCIONES DE DESPLIEGUE (RESUMEN)

### Pasos para levantar el entorno:

#### Paso 1: Preparación del Clúster

- Crear clúster en GKE.
- Habilitar la virtualización anidada (necesaria para KubeVirt).
- Instalar el operador de KubeVirt y el operador de Strimzi.

#### Paso 2: Construcción de Imágenes

Compilar y subir las imágenes de Docker a su Container Registry (Zot o GCR):
- api-rust
- grpc-server-go
- kafka-consumer-go
- valkey-custom-disk (Imagen de disco QCOW2 empaquetada en Docker).

#### Paso 3: Aplicar Manifiestos

Ejecutar los siguientes comandos:

```bash
kubectl create namespace blackfriday
# 1. Infraestructura Base
kubectl apply -f k8s/strimzi-kafka.yaml
kubectl apply -f k8s/kafka-topic.yaml
# 2. Base de Datos Virtualizada
kubectl apply -f k8s/valkey-vm.yaml
# 3. Aplicaciones y Configuración de Red
kubectl apply -f k8s/apps.yaml
kubectl apply -f k8s/ingress.yaml 
kubectl apply -f k8s/hpa.yaml      
# 4. Consumidor y Monitoreo
kubectl apply -f k8s/consumer.yaml
kubectl apply -f k8s/grafana.yaml
```

#### Paso 4: Ejecución de Pruebas (Locust)

- Iniciar Locust localmente: `locust -f locustfile.py`
- Acceder a http://localhost:8089.
- Configurar Host con la IP externa del servicio api-rust-service.
- Iniciar prueba con 200 usuarios y tasa de 10 usuarios/segundo.

## SECCIÓN 6: DESARROLLO Y RETOS ENCONTRADOS

### Proceso de Desarrollo

El desarrollo se realizó de forma iterativa. Primero se estableció la comunicación básica Rust-Go, luego se integró Kafka, y finalmente se abordó la complejidad de KubeVirt.

### Retos Principales y Soluciones:

#### Reto 1: Configuración de Valkey en KubeVirt

**Problema**: La VM iniciaba pero Valkey no aceptaba conexiones externas.

**Solución**: Se modificó el archivo de configuración /etc/valkey.conf dentro de la imagen Alpine para establecer bind 0.0.0.0 y desactivar el modo protegido. Se creó una imagen de disco personalizada (QCOW2) con estos cambios persistentes.

#### Reto 2: Visualización de Datos de Texto en Grafana

**Problema**: Grafana mostraba los valores numéricos (Score) en lugar de los nombres de los productos en los paneles de "Producto más vendido".

**Solución**: Se aplicaron transformaciones en Grafana para ocultar el campo Value y se configuró el panel Stat en modo "Text Mode: Name", forzando la visualización del miembro del Sorted Set.

#### Reto 3: Persistencia de Datos

**Problema**: Al reiniciar los pods, los datos en memoria se perdían.

**Solución**: La implementación de KubeVirt permite que la base de datos corra en un entorno persistente simulado, y el uso de Kafka asegura que, incluso si el consumidor cae, los mensajes quedan en cola (Backpressure) hasta que el servicio se recupera.



## SECCIÓN 7: ANÁLISIS DE RENDIMIENTO Y RÉPLICAS


### ANÁLISIS DE RENDIMIENTO: IMPACTO DE LAS RÉPLICAS

Como parte de los requerimientos, se realizó una prueba comparativa variando el número de réplicas de los consumidores de Go (Writers) encargados de procesar la cola de Kafka y escribir en Valkey.

### Escenario 1: 1 Réplica

Con una sola instancia del consumidor procesando mensajes, se observó que bajo una carga de 200 usuarios concurrentes en Locust, el "Lag" (retraso) en Kafka aumentaba progresivamente. El consumidor lograba procesar aproximadamente 80-100 mensajes por segundo, pero la cola de mensajes pendientes crecía, lo que resultaba en un retraso de hasta 5 segundos entre que la venta ocurría y se reflejaba en Grafana.

### Escenario 2: 2 Réplicas (Escalado Horizontal)

Al aumentar a 2 réplicas (o permitir que el HPA escalara automáticamente), el rendimiento de procesamiento se duplicó efectivamente. El throughput de escritura en Valkey subió a ~180 mensajes por segundo.

### Conclusión

El diseño "Stateless" del consumidor en Go permite un escalado lineal casi perfecto. Al haber múltiples consumidores en el mismo "Consumer Group" de Kafka, los mensajes se reparten automáticamente entre los pods, eliminando el cuello de botella y reduciendo la latencia de visualización en Grafana a tiempo casi real (<1 segundo).


## SECCIÓN 8: CONCLUSIONES Y COMPARATIVAS

### Rendimiento API REST vs gRPC

Se observó que la comunicación interna entre Rust y Go mediante gRPC es significativamente más eficiente que REST. gRPC utiliza Protocol Buffers (binario), lo que reduce el tamaño del payload y la latencia de serialización comparado con JSON, algo crítico en sistemas de alto tráfico.

### Rol de Kafka y Valkey

Kafka demostró ser fundamental para la resiliencia. Durante las pruebas de carga con Locust, Kafka actuó como un buffer, absorbiendo picos de tráfico que la base de datos no podría haber manejado en tiempo real mediante inserción directa. Valkey, al ser una base de datos en memoria, permitió tiempos de lectura de sub-milisegundos para que Grafana actualizara los dashboards en tiempo real.

### Impacto de la Virtualización (KubeVirt)

Integrar una VM dentro de Kubernetes añade una capa de complejidad pero ofrece flexibilidad. Permite ejecutar cargas de trabajo que requieren un SO completo (como una configuración específica de base de datos legacy o personalizada) sin salir del ecosistema de orquestación de contenedores.
