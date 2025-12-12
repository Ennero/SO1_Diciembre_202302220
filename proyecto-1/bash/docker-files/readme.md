# Dockerfiles de prueba

Construya las imágenes base usadas por los scripts y el daemon.

## Imágenes

- `dockerfile.ram`: carga de memoria (ej. `stress-ng` sobre RAM)
- `dockerfile.cpu`: carga de CPU (ej. `stress-ng` sobre CPU)
- `dockerfile.low`: carga baja (ej. procesos ligeros tipo `sleep`)

## Construcción

Ejecutar desde la carpeta `proyecto-1/` del repositorio:

```bash
# Construir imagen de alto consumo de RAM
docker build -t so1_ram -f docker-files/dockerfile.ram .

# Construir imagen de alto consumo de CPU
docker build -t so1_cpu -f docker-files/dockerfile.cpu .

# Construir imagen de bajo consumo
docker build -t so1_low -f docker-files/dockerfile.low .
```

## Verificación

```bash
docker images | grep -E "so1_(ram|cpu|low)"
```
