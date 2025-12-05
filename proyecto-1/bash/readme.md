# Script `generator.sh`

Crea 10 contenedores aleatorios a partir de las imágenes `so1_ram`, `so1_cpu` y `so1_low`, asignando nombres únicos para facilitar el rastreo.

## Requisitos

- Imágenes construidas previamente (ver: [`../docker-files/`](../docker-files/))

## Ejecución

```bash
# Dar permisos de ejecución (opcional)
chmod +x generator.sh

# Ejecutar
./generator.sh

# Alternativa sin cambiar permisos
bash generator.sh
```

## Notas

- Los contenedores se nombran como `so1_contenedor_<imagen>_<random>`.
- Para detener y eliminar contenedores de prueba:
	```bash
	docker ps -aq --filter name=so1_contenedor_ | xargs -r docker stop
	docker ps -aq --filter name=so1_contenedor_ | xargs -r docker rm
	```

