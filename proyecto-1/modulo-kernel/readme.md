# Modulo de Kernel

## Creación de Módulo
Para poder crear este módulo, primero es necesario descargar el compilador gcc y sus herramientas escenciales:

```bash
sudo apt update
sudo apt install build-essential
```

Además es recomendable asegurarse de tener el encabezado del kernel instalado:

```bash
sudo apt-get update
sudo apt-get install build-essential linux-headers-$(uname -r)
```

Ya con esto, ingresando el siguiente comando:

```bash
make
```

Se crearía el módulo.



## Probar módulo

Para esto se usa el siguiente comando:

```bash
sudo insmod module.ko
```
