# Manifiestos de Kubernetes

Descripción breve de los archivos en esta carpeta. Todas las referencias a direcciones o hostnames sensibles se expresan con marcadores `<...>`.

## Archivos
- `apps.yaml`: namespace `blackfriday`, Deployments/Services de `grpc-server-go` y `api-rust`.
- `go-client.yaml`: Deployment y Service para el cliente HTTP–gRPC.
- `consumer.yaml`: Deployment del consumidor de Kafka.
- `valkey-vm.yaml`: VirtualMachine KubeVirt y Service `valkey-service` en 6379.
- `grafana.yaml`: Deployment y Service tipo LoadBalancer para Grafana (puerto 80 -> 3000).
- `ingress.yaml`: reglas NGINX para enviar `/` a `api-rust-service`.
- `strimzi-kafka.yaml`: clúster Kafka (KRaft) gestionado por Strimzi.
- `kafka-topic.yaml`: tópico `ventas`.
- `hpa.yaml`: HPA para `api-rust`.

## Notas
- Sustituye `<direccion IP del registry privado>` por tu registry real antes de desplegar.
- Usa `kubectl apply -f <archivo>` desde esta carpeta o con rutas relativas.
- Mantén los valores sensibles (IPs, credenciales) fuera del repositorio; usa ConfigMaps/Secrets o variables de entorno.
