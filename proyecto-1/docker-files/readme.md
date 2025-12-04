# Crear imágenes

Con los archivos dockerfile, se crean las imágenes para las instancias:

```bash
# Construir imagen de alto consumo de RAM
docker build -t so1_ram -f docker-files/dockerfile.ram .

# Construir imagen de alto consumo de CPU
docker build -t so1_cpu -f docker-files/dockerfile.cpu .

# Construir imagen de bajo consumo
docker build -t so1_low -f docker-files/dockerfile.low .
```