# Módulo de Kernel

Expone métricas de procesos en `/proc/continfo_so1_202302220` en formato JSON.

## Requisitos

```bash
sudo apt update
sudo apt install -y build-essential linux-headers-$(uname -r)
```

## Compilación

```bash
make clean && make
```

Genera `module.ko` usando el `Makefile` (objetivo `obj-m += module.o`).

Archivos principales:

- `module.c` — lógica del procfs (JSON) — [ver archivo](./module.c)
- `Makefile` — reglas de compilación — [ver archivo](./Makefile)

## Carga y verificación

```bash
sudo insmod module.ko
cat /proc/continfo_so1_202302220
```

Salida esperada: arreglo JSON con `pid`, `name`, `state`, `ram_kb`, `vsz_kb`, `cpu_utime`, `cpu_stime`.

## Descarga

```bash
sudo rmmod module
```

## Notas

- La entrada `/proc` es de solo lectura (0444).
- Si `insmod` falla, revise `dmesg | tail -n 50`.
