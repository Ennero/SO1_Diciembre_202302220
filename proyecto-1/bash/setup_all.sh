#!/usr/bin/env bash
set -euo pipefail

# ==========================================
# SO1 Proyecto 1 - Setup Robusto (v2.1)
# ==========================================

# --- CONFIGURACIÓN ---
PROJECT_DIR_NAME="proyecto-1"

# Detectar el usuario real detrás de 'sudo'
if [[ -n "${SUDO_USER:-}" ]]; then
    REAL_USER="$SUDO_USER"
    REAL_HOME=$(getent passwd "$SUDO_USER" | cut -d: -f6)
else
    REAL_USER="$USER"
    REAL_HOME="$HOME"
fi

# Definimos DÓNDE debería estar el proyecto (La verdad absoluta)
TARGET_DIR="$REAL_HOME/$PROJECT_DIR_NAME"

# --- COLORES Y LOGS ---
log() { echo -e "\n🟢 [INFO] $1"; }
warn() { echo -e "\n🟡 [WARN] $1"; }
err() { echo -e "\n🔴 [ERROR] $1" >&2; }
die() { echo -e "\n❌ [FATAL] $1" >&2; exit 1; }

# 1. VERIFICAR PERMISOS ROOT
if [[ $EUID -ne 0 ]]; then
  die "Este script debe ejecutarse con sudo: sudo ./setup_all.sh"
fi

# 2. MIGRACIÓN Y DETECCIÓN DE RUTA (CORREGIDO)
migrate_project() {
  log "Verificando ubicación del proyecto..."
  
  # CORRECCIÓN CLAVE:
  # Si estamos en cualquier subcarpeta de /home/usuario/proyecto-1 (ej: .../bash),
  # forzamos a que la ruta del proyecto sea la RAÍZ, no la subcarpeta.
  
  if [[ "$PWD" == "$TARGET_DIR"* ]]; then
    log "Estás dentro de la estructura del proyecto."
    # Forzamos la ruta a la raíz definida, ignorando si estás en /bash
    PROJECT_PATH="$TARGET_DIR"
    log "Raíz del proyecto fijada en: $PROJECT_PATH"
    return
  fi

  # Caso B: Estamos en carpeta compartida o externa (/mnt, /media, etc)
  log "Entorno externo detectado ($PWD). Migrando a: $TARGET_DIR"

  if [[ ! -d "$TARGET_DIR" ]]; then
    mkdir -p "$TARGET_DIR"
    log "Carpeta creada."
  else
    warn "La carpeta destino ya existe. Sincronizando archivos..."
  fi

  # Copiar contenido
  # Nota: Copiamos desde el directorio PADRE si estamos corriendo el script dentro de una carpeta 'bash' suelta
  # Pero asumiremos copia recursiva estándar del directorio actual.
  cp -r ./* "$TARGET_DIR/" || true
  
  # Corregir permisos
  chown -R "$REAL_USER:$REAL_USER" "$TARGET_DIR"
  
  PROJECT_PATH="$TARGET_DIR"
  log "✅ Proyecto migrado exitosamente."
}

# 3. INSTALAR HERRAMIENTAS
install_dependencies() {
  log "Verificando dependencias..."
  apt-get update -qq
  DEPS="build-essential linux-headers-$(uname -r) docker.io golang make gcc"
  log "Instalando: $DEPS"
  apt-get install -y $DEPS
  systemctl enable --now docker || true
  usermod -aG docker "$REAL_USER" || true
}

# 4. CONSTRUIR IMÁGENES
build_docker_images() {
  log "Preparando imágenes Docker..."
  
  # Aseguramos ir a la RAÍZ del proyecto
  cd "$PROJECT_PATH"

  if [[ ! -d "docker-files" ]]; then
     # Intento de autocuración por si se ejecutó desde 'bash' sin migrar
     if [[ -d "../docker-files" ]]; then
        cd ..
        PROJECT_PATH="$PWD"
     else
        die "No se encuentra la carpeta 'docker-files' en $PROJECT_PATH. Verifica tu estructura."
     fi
  fi

  log "Construyendo imágenes desde: $PWD"
  docker build -t so1_ram -f docker-files/dockerfile.ram . > /dev/null
  docker build -t so1_cpu -f docker-files/dockerfile.cpu . > /dev/null
  docker build -t so1_low -f docker-files/dockerfile.low . > /dev/null
  log "✅ Imágenes construidas."
}

# 5. KERNEL
kernel_setup() {
  log "Configurando Módulos del Kernel..."
  cd "$PROJECT_PATH/modulo-kernel"

  make clean > /dev/null
  make > /dev/null || die "Error compilando los módulos C."

  rmmod procesos 2>/dev/null || true
  rmmod ram 2>/dev/null || true

  insmod procesos.ko || die "Fallo al insertar procesos.ko"
  insmod ram.ko || die "Fallo al insertar ram.ko"
  
  if lsmod | grep -q "so1"; then
     log "✅ Módulos cargados correctamente."
  fi
}

# 6. DAEMON
launch_daemon() {
  log "Preparando entorno del Daemon..."
  cd "$PROJECT_PATH/go-daemon"

  touch metrics.db
  chmod 666 metrics.db
  chmod +x ../bash/generator.sh

  if [ ! -f "go.mod" ]; then
    go mod init daemon 2>/dev/null || true
  fi
  go mod tidy

  log "🚀 EJECUTANDO SISTEMA..."
  echo "---------------------------------------------------"
  env "PATH=$PATH" go run main.go
}

# --- MAIN ---
main() {
  migrate_project
  install_dependencies
  build_docker_images
  kernel_setup
  launch_daemon
}

main "$@"