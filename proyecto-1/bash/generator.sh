#!/bin/bash

# Nombres de las imagenes que creamos
IMAGES=("so1_ram" "so1_cpu" "so1_low")

# Generar 10 contenedores
for i in {1..10}; do
    # Seleccionar un índice aleatorio entre 0 y 2
    RANDOM_INDEX=$((RANDOM % 3))
    IMAGE_NAME=${IMAGES[$RANDOM_INDEX]}
    
    # Crear un nombre único para el contenedor para poder rastrearlo
    CONTAINER_NAME="so1_contenedor_${IMAGE_NAME}_${RANDOM}"

    echo "Creando contenedor: $CONTAINER_NAME con imagen $IMAGE_NAME"
    
    # Ejecutar en modo detached (-d)
    docker run -d --name "$CONTAINER_NAME" "$IMAGE_NAME"
done