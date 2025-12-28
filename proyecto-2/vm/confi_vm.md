# Configuración de la VM y registry (sanitizado)

Pasos mínimos para preparar la VM que aloja el registry Zot. Las direcciones IP se expresan como marcadores.

## Instalar Docker
```bash
sudo apt-get update
sudo apt-get install -y docker.io
sudo systemctl start docker
sudo systemctl enable docker
sudo usermod -aG docker $USER
```

## Configurar Zot (registry privado)
```bash
mkdir -p zot-registry && cd zot-registry
cat > config.json <<'EOF'
{
  "distSpecVersion": "1.1.0",
  "storage": {
    "rootDirectory": "/var/lib/zot/data"
  },
  "http": {
    "address": "<direccion IP de escucha>",
    "port": "5000"
  },
  "log": {
    "level": "debug"
  }
}
EOF

docker run -d \
  --name zot \
  -p 5000:5000 \
  -v $(pwd)/config.json:/etc/zot/config.json \
  -v $(pwd)/data:/var/lib/zot/data \
  ghcr.io/project-zot/zot-linux-amd64:latest
```

## Firewall (GCP)
- Origen permitido: `<rango IP autorizado>` (evitar `<rango IP abierto>` en producción).
- Puerto: TCP 5000.

## Probar
```bash
curl http://<direccion IP del registry privado>:5000/v2/_catalog
```
Debe responder un JSON con los repositorios disponibles.
