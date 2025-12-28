FROM rust:1.83-slim-bookworm AS builder
WORKDIR /app

# Instalar dependencias del sistema necesarias para compilar (SSL, pkg-config)
RUN apt-get update && apt-get install -y pkg-config libssl-dev && rm -rf /var/lib/apt/lists/*

# Copiar el código fuente (estamos en la carpeta api-rust, así que copiamos todo a /app)
COPY . .

# Compilar el binario en modo release
RUN cargo build --release

FROM debian:bookworm-slim
WORKDIR /app

# Instalar certificados y librerías SSL para ejecución
RUN apt-get update && apt-get install -y ca-certificates libssl3 && rm -rf /var/lib/apt/lists/*

# COPIAR EL BINARIO
COPY --from=builder /app/target/release/api-rust .

# Exponer puerto
EXPOSE 8080

# Ejecutar
CMD ["./api-rust"]