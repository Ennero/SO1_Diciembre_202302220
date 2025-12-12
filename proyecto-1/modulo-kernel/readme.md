# Módulos de Kernel

Expone métricas en `/proc` en formato JSON desde dos módulos: procesos del sistema y métricas de memoria/contenedores.

## Requisitos

```bash
sudo apt update
sudo apt install -y build-essential linux-headers-$(uname -r)
```

## Compilación

```bash
make clean && make
```

Genera `procesos.ko` y `ram.ko` usando el `Makefile`.

Archivos principales:

- `procesos.c` — procfs de procesos — [ver archivo](./procesos.c)
- `ram.c` — procfs de memoria RAM — [ver archivo](./ram.c)
- `Makefile` — reglas de compilación — [ver archivo](./Makefile)

## Carga y verificación

```bash
sudo insmod procesos.ko
sudo insmod ram.ko

# Verificar entradas /proc
cat /proc/sysinfo_so1_202302220
cat /proc/continfo_so1_202302220
```

Salida esperada:
- `sysinfo_so1_202302220`: arreglo JSON con `pid`, `name`, `state`, `ram_kb`, `vsz_kb`, `cpu_utime`, `cpu_stime`.
- `continfo_so1_202302220`: objeto JSON con `total_ram_mb`, `free_ram_mb`, `used_ram_mb`, `percentage`.

## Descarga

```bash
sudo rmmod procesos
sudo rmmod ram
```

## Notas

- Las entradas `/proc` son de solo lectura (`0444`).
- Si `insmod` falla, revise `dmesg | tail -n 50`.
