# Script `generator.sh`

Este script automatiza la creación de contenedores. Genera 10 contenedores aleatorios a partir de las imágenes `so1_ram`, `so1_cpu` y `so1_low`.

## Requisitos previa ejecución

- Imágenes de docker construidas previamente.
- Permisos de ejecución en el script:
  ```bash
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

