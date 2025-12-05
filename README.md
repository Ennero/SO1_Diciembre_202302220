# SO1_Diciembre_202302220

Proyecto de Sistemas Operativos 1: monitoreo de contenedores con módulo de Kernel (C) y daemon (Go), automatizado con Bash y contenedores Docker.

## Contenidos

- `proyecto-1/modulo-kernel/`: módulo de Kernel que expone métricas en `/proc` — [ver carpeta](proyecto-1/modulo-kernel/)
- `proyecto-1/go-daemon/`: daemon en Go que cruza métricas y gestiona procesos — [ver carpeta](proyecto-1/go-daemon/)
- `proyecto-1/bash/`: scripts para generar carga — [ver carpeta](proyecto-1/bash/)
- `proyecto-1/docker-files/`: Dockerfiles para imágenes de prueba — [ver carpeta](proyecto-1/docker-files/)
- `proyecto-1/documentacion/`: manuales técnico y de usuario — [ver carpeta](proyecto-1/documentacion/)

## Inicio Rápido

Consulte los manuales:

- Manual de Usuario — [documentacion/manual_usuario.md](proyecto-1/documentacion/manual_usuario.md)
- Manual Técnico — [documentacion/manual_tecnico.md](proyecto-1/documentacion/manual_tecnico.md)

Comandos esenciales (desde `proyecto-1/`):

```bash
# Construir imágenes
docker build -t so1_ram -f docker-files/dockerfile.ram .
docker build -t so1_cpu -f docker-files/dockerfile.cpu .
docker build -t so1_low -f docker-files/dockerfile.low .

# Compilar y cargar módulo
cd modulo-kernel && make && sudo insmod module.ko && cd -
cat /proc/continfo_so1_202302220

# Generar carga
cd bash && chmod +x generator.sh && ./generator.sh && cd -

# Ejecutar daemon
cd go-daemon && sudo env "PATH=$PATH" go run main.go
```

## Licencia

Uso académico. Consulte archivos fuente para detalles.
