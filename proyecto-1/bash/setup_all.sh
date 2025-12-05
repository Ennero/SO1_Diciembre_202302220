#!/usr/bin/env bash
set -euo pipefail

# SO1 Proyecto 1 - Instalación y Preparación (Optimizado para Daemon Go)
# 1. Mueve el proyecto al Home (para evitar errores de SQLite/Docker en carpetas compartidas).
# 2. Instala dependencias y compila módulos.
# 3. Construye imágenes de Docker.
# 4. Cede el control al Daemon de Go.

# --- CONFIGURACIÓN DE RUTAS ---
# Ajusta esto si tu carpeta compartida tiene otro nombre
SHARED_MOUNTPOINT="/mnt/compartido"   
PROJECT_DIR_NAME="proyecto-1"         
ROOT_DIR="$HOME"                      

log() { echo -e "\n🟢 [INFO] $1"; }
warn() { echo -e "\n🟡 [WARN] $1"; }
err() { echo -e "\n🔴 [ERROR] $1" >&2; }

# 1. VERIFICAR PERMISOS
if [[ $EUID -ne 0 ]]; then
  warn "Este script debe ejecutarse con sudo para instalar paquetes y cargar módulos."
  warn "Ejecuta: sudo ./setup_all.sh"
  exit 1
fi

# 2. MIGRAR PROYECTO (Vital para evitar el error 'database is locked')
migrate_project_to_home() {
  log "Verificando ubicación del proyecto..."
  
  # Si ya estamos en el home, no hacemos nada
  if [[ "$PWD" == "$HOME/"* ]]; then
    log "Ya estás ejecutando desde el Home. Continuando..."
    PROJECT_PATH="$PWD"
    return
  fi

  # Si estamos en /mnt (carpeta compartida), copiamos
  log "Detectado entorno de carpeta compartida. Copiando a $ROOT_DIR para evitar errores de permisos..."
  
  TARGET_DIR="$ROOT_DIR/$PROJECT_DIR_NAME"
  
  if [[ -d "$TARGET_DIR" ]]; then
    warn "La carpeta $TARGET_DIR ya existe. Actualizando archivos..."
    cp -r ./* "$TARGET_DIR/"
  else
    mkdir -p "$TARGET_DIR"
    cp -r ./* "$TARGET_DIR/"
  fi
  
  # Ajustar permisos para que tu usuario (no root) sea el dueño, pero root pueda ejecutar
  # Asumimos que el usuario es el que invocó sudo (SUDO_USER)
  if [[ -n "${SUDO_USER:-}" ]]; then
      chown -R "$SUDO_USER:$SUDO_USER" "$TARGET_DIR"
  fi

  PROJECT_PATH="$TARGET_DIR"
  log "Proyecto preparado en: $PROJECT_PATH"
}

# 3. INSTALAR HERRAMIENTAS
install_dependencies() {
  log "Instalando dependencias (Go, Docker, GCC, Make)..."
  apt-get update -qq
  apt-get install -y build-essential linux-headers-$(uname -r) docker.io docker-compose golang make gcc

  systemctl enable --now docker || true
}

# 4. CONSTRUIR IMÁGENES (Requisito para que el Go/Script funcione)
build_docker_images() {
  log "Construyendo imágenes Docker (so1_ram, so1_cpu, so1_low)..."
  cd "$PROJECT_PATH"
  
  # Usamos los Dockerfiles que están en la raíz o carpeta docker-files
  docker build -t so1_ram -f docker-files/dockerfile.ram .
  docker build -t so1_cpu -f docker-files/dockerfile.cpu .
  docker build -t so1_low -f docker-files/dockerfile.low .
}

# 5. MÓDULOS DEL KERNEL
build_and_load_kernel_modules() {
  log "Compilando y cargando módulos del Kernel..."
  cd "$PROJECT_PATH/modulo-kernel"
  
  make clean && make
  
  # Descargar por si ya estaban cargados (evita error "File exists")
  rmmod procesos 2>/dev/null || true
  rmmod ram 2>/dev/null || true

  insmod procesos.ko
  insmod ram.ko
  
  log "Verificando lectura de /proc..."
  # Usamos continfo como definiste en tu código C
  if cat /proc/continfo_so1_202302220 > /dev/null; then
      log "Módulo RAM: OK"
  else 
      err "Fallo al leer continfo_so1..."
  fi
}

# 6. EJECUTAR DAEMON
run_daemon() {
  log "Preparando ejecución del Daemon..."
  cd "$PROJECT_PATH/go-daemon"
  
  # Asegurar que el script generador tenga permisos (CRÍTICO para tu automatización)
  chmod +x ../bash/generator.sh
  
  # Instalar dependencias de Go
  go mod tidy

  log "🚀 INICIANDO SISTEMA COMPLETO..."
  log "El Daemon levantará Grafana y generará tráfico automáticamente."
  log "Presiona Ctrl+C para detener todo."
  
  # Ejecutar Go pasando el PATH para que encuentre Docker
  env "PATH=$PATH" go run main.go
}

# --- FLUJO PRINCIPAL ---
main() {
  migrate_project_to_home
  install_dependencies
  build_docker_images
  build_and_load_kernel_modules
  run_daemon
}

main "$@"