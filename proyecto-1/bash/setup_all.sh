#!/usr/bin/env bash
set -euo pipefail

# SO1 Proyecto 1 - Instalación y Ejecución Automática
# Este script prepara dependencias, migra desde carpeta compartida a Home,
# construye imágenes, compila módulos, levanta Grafana y ejecuta el daemon.

# Configuración
SHARED_TARGET_NAME="micarpeta"        # Nombre configurado en Virt-Manager como Target path
SHARED_MOUNTPOINT="/mnt/compartido"   # Punto de montaje dentro de la VM
PROJECT_DIR_NAME="proyecto-1"         # Nombre del proyecto dentro de la carpeta compartida
ROOT_DIR="$HOME"                      # Directorio destino para la copia

log() { echo -e "\n[INFO] $1"; }
warn() { echo -e "\n[WARN] $1"; }
err() { echo -e "\n[ERROR] $1" >&2; }

require_sudo() {
  if [[ $EUID -ne 0 ]]; then
    warn "Se requieren algunos comandos con sudo. Se te pedirá contraseña cuando corresponda."
  fi
}

ensure_shared_mount() {
  log "Montando carpeta compartida en $SHARED_MOUNTPOINT (virtiofs: $SHARED_TARGET_NAME)"
  sudo mkdir -p "$SHARED_MOUNTPOINT"
  if mountpoint -q "$SHARED_MOUNTPOINT"; then
    log "Ya está montado: $SHARED_MOUNTPOINT"
  else
    if sudo mount -t virtiofs "$SHARED_TARGET_NAME" "$SHARED_MOUNTPOINT"; then
      log "Montaje virtiofs exitoso."
    else
      warn "virtiofs no disponible o falló. Intentando con 9p..."
      sudo mount -t 9p -o trans=virtio,version=9p2000.L "$SHARED_TARGET_NAME" "$SHARED_MOUNTPOINT"
      log "Montaje 9p exitoso."
    fi
  fi
}

migrate_project_to_home() {
  log "Migrando proyecto desde carpeta compartida a Home"
  cd "$ROOT_DIR"
  if [[ -d "$PROJECT_DIR_NAME" ]]; then
    warn "Ya existe $PROJECT_DIR_NAME en Home. Creando copia alternativa: ${PROJECT_DIR_NAME}-compartido"
    cp -r "$SHARED_MOUNTPOINT/$PROJECT_DIR_NAME" "${PROJECT_DIR_NAME}-compartido"
    PROJECT_PATH="$ROOT_DIR/${PROJECT_DIR_NAME}-compartido"
  else
    cp -r "$SHARED_MOUNTPOINT/$PROJECT_DIR_NAME" .
    PROJECT_PATH="$ROOT_DIR/$PROJECT_DIR_NAME"
  fi
  log "Proyecto copiado en: $PROJECT_PATH"
}

install_dependencies() {
  log "Instalando dependencias del sistema (apt)"
  sudo apt update
  sudo apt install -y build-essential linux-headers-$(uname -r) docker.io docker-compose golang make gcc

  # Alternativa por Snap (opcional)
  if ! command -v docker >/dev/null 2>&1; then
    warn "Docker no encontrado tras apt. Intentando instalación vía Snap."
    sudo snap install docker || warn "Falló instalación vía Snap. Continúo si docker aparece luego."
  fi

  log "Habilitando y arrancando Docker"
  sudo systemctl enable --now docker || true

  log "Añadiendo usuario actual al grupo docker (opcional)"
  sudo usermod -aG docker "$USER" || true
  warn "Si es la primera vez que se agrega al grupo docker, cierre sesión y vuelva a entrar."

  log "Verificaciones rápidas"
  docker --version || warn "docker no responde"
  docker-compose --version || warn "docker-compose no responde"
  go version || warn "go no responde"
  gcc --version || true
  make --version || true
}

build_docker_images() {
  log "Construyendo imágenes Docker"
  cd "$PROJECT_PATH"
  sudo docker build -t so1_ram -f docker-files/dockerfile.ram .
  sudo docker build -t so1_cpu -f docker-files/dockerfile.cpu .
  sudo docker build -t so1_low -f docker-files/dockerfile.low .
}

build_and_load_kernel_modules() {
  log "Compilando y cargando módulos del kernel"
  cd "$PROJECT_PATH/modulo-kernel"
  make clean && make
  sudo insmod procesos.ko || warn "insmod procesos.ko falló; revise dmesg"
  sudo insmod ram.ko || warn "insmod ram.ko falló; revise dmesg"
  log "Verificando /proc"
  cat /proc/sysinfo_so1_202302220 || warn "No se puede leer sysinfo"
  cat /proc/continfo_so1_202302220 || warn "No se puede leer continfo"
}

start_grafana_stack() {
  log "Levantando Grafana via docker-compose"
  cd "$PROJECT_PATH"
  touch go-daemon/metrics.db
  chmod 666 go-daemon/metrics.db
  cd dashboard
  sudo docker-compose up -d
  log "Grafana arriba en http://localhost:3000 (admin/admin)"
}

run_daemon() {
  log "Ejecutando daemon de Go"
  cd "$PROJECT_PATH/go-daemon"
  go mod tidy
  sudo env "PATH=$PATH" go run main.go
}

main() {
  require_sudo
  ensure_shared_mount
  migrate_project_to_home
  install_dependencies
  build_docker_images
  build_and_load_kernel_modules
  start_grafana_stack
  run_daemon
}

main "$@"
