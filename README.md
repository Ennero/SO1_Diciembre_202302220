# SO1_Diciembre_202302220

Proyecto de Sistemas Operativos 1: monitoreo de contenedores con módulos de Kernel (C) y daemon (Go), automatizado con Bash y contenedores Docker.

## Contenidos

- `proyecto-1/modulo-kernel/`: módulos de Kernel que exponen métricas en `/proc` — [ver carpeta](proyecto-1/modulo-kernel/)
- `proyecto-1/go-daemon/`: daemon en Go que cruza métricas y gestiona procesos — [ver carpeta](proyecto-1/go-daemon/)
- `proyecto-1/bash/`: scripts para generar carga — [ver carpeta](proyecto-1/bash/)
- `proyecto-1/docker-files/`: Dockerfiles para imágenes de prueba — [ver carpeta](proyecto-1/docker-files/)
- `proyecto-1/documentacion/`: manuales técnico y de usuario — [ver carpeta](proyecto-1/documentacion/)

## Inicio Rápido

Consulte los manuales:

- Manual de Usuario — [documentacion/manual_usuario.md](proyecto-1/documentacion/manual_usuario.md)
- Manual Técnico — [documentacion/manual_tecnico.md](proyecto-1/documentacion/manual_tecnico.md)

### Ejecución automática (recomendada)

```bash
cd proyecto-1/bash
chmod +x setup_all.sh
sudo ./setup_all.sh
```

- Migra el proyecto desde carpeta compartida a `~/proyecto-1` (evita locking/ACL en SQLite/Docker).
- Instala dependencias, construye imágenes, compila/carga módulos del kernel.
- El daemon de Go levanta Grafana automáticamente y genera tráfico periódico.
- Acceso Grafana: http://localhost:3000 (admin/admin).

### Ejecución manual (resumen)

```bash
# Dependencias
sudo apt update
sudo apt install -y build-essential linux-headers-$(uname -r) docker.io docker-compose golang
sudo systemctl enable --now docker

# Migrar desde carpeta compartida (si aplica)
cd ~ && cp -r /mnt/compartido/proyecto-1 . && cd proyecto-1

# Imágenes Docker
sudo docker build -t so1_ram -f docker-files/dockerfile.ram .
sudo docker build -t so1_cpu -f docker-files/dockerfile.cpu .
sudo docker build -t so1_low -f docker-files/dockerfile.low .

# Módulos de kernel
cd modulo-kernel && make clean && make && sudo insmod procesos.ko && sudo insmod ram.ko && cd -

# Verificar /proc
cat /proc/sysinfo_so1_202302220
cat /proc/continfo_so1_202302220

# Grafana
touch go-daemon/metrics.db && chmod 666 go-daemon/metrics.db
cd dashboard && sudo docker-compose up -d && cd -

# Daemon Go
cd go-daemon && go mod tidy && sudo env "PATH=$PATH" go run main.go
```

## Licencia

Uso académico. Consulte archivos fuente para detalles.
