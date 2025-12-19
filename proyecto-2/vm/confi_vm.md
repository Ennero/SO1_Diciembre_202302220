# Configuración de Máquina Virtual

## Instalación de Docker en la VM

```bash
# 1. Actualizar repositorios
sudo apt-get update

# 2. Instalar Docker
sudo apt-get install -y docker.io

# 3. Iniciar el servicio y habilitarlo
sudo systemctl start docker
sudo systemctl enable docker

# 4. Dar permisos a tu usuario (para no usar sudo en cada comando docker)
sudo usermod -aG docker $USER
```

## Configurar y Ejecutar ZOT

```bash
# 1. Crear Carpeta para ZOT

mkdir -p zot-registry
cd zot-registry

# 2. Crea un archivo llamado config.json
nano config.json

```

Dentro del editor de texto, se pega lo siguiente:

```json
{
  "distSpecVersion": "1.1.0",
  "storage": {
    "rootDirectory": "/var/lib/zot/data"
  },
  "http": {
    "address": "0.0.0.0",
    "port": "5000"
  },
  "log": {
    "level": "debug"
  }
}
```
(Guarda con `Ctrl+O`, Enter y sal con `Ctrl+X`)

Luego se Ejecuta el contenedor Zot:

```bash
docker run -d \
  --name zot \
  -p 5000:5000 \
  -v $(pwd)/config.json:/etc/zot/config.json \
  -v $(pwd)/data:/var/lib/zot/data \
  ghcr.io/project-zot/zot-linux-amd64:latest
```

## Abrir Firewall de Google Cloud


1. Ir a la consola de GCP en el navegador.

2. Buscar "VPC network" (Red de VPC) -> "Firewall".

3. Hacer clic en "Create Firewall Rule" (Crear regla de firewall).
   1. Name: allow-zot-5000
   2. Targets: All instances in the network (Todas las instancias).
   3. Source IPv4 ranges: 0.0.0.0/0 (Esto permite acceso desde cualquier lado, útil para desarrollo).
   4. Protocols and ports: Marca "TCP" y escribe 5000.

4. Darle a Create.


## Probar

Intentar ver el catálogo de registry usando la IP externa de la VM

```bash
curl http://<IP-EXTERNA-DE-TU-VM>:5000/v2/_catalog
```

Si te devuelve algo como ``{"repositories":[]}``.
