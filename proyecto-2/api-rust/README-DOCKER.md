# Dockerización (Rust)

Build multi-etapa para la API Rust. Todo identificador sensible se sustituyó por marcadores.

## Consideraciones
- Objetivo: binario estático en imagen final Debian slim.
- Dependencias: `pkg-config` y `libssl-dev` en la etapa de build para `reqwest`.
- El código ya no compila protos; `build.rs` está vacío.

## Comandos (desde la raíz `proyecto-2/`)
```bash
docker build -f api-rust/Dockerfile -t <direccion IP del registry privado>:5000/api-rust:v3 .
docker push <direccion IP del registry privado>:5000/api-rust:v3
```

## Problemas resueltos
- Forzado `edition = "2021"` para evitar `edition2024` inestable.
- Actualización de dependencias conflictivas con `cargo update` (ej.: `home 0.5.9`).

## Nota de seguridad
No publiques imágenes con IPs incrustadas; usa variables o placeholders y configura el registry en una red segura.


