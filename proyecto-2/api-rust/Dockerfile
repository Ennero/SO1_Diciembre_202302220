# Compilación
FROM rust:1.83-slim-bookworm AS builder
WORKDIR /usr/src/proyecto

# Instalar compilador de Protobuf 
RUN apt-get update && apt-get install -y protobuf-compiler libprotobuf-dev

# Copiar todo el contexto
COPY . .

# Moverse a la carpeta de la API y compilar
WORKDIR /usr/src/proyecto/api-rust
RUN cargo build --release

# Imagen final
FROM debian:bookworm-slim
WORKDIR /app

# Instalar librerías mínimas (SSL/TLS)
RUN apt-get update && apt-get install -y ca-certificates libssl3 && rm -rf /var/lib/apt/lists/*

# Copiar el binario compilado
COPY --from=builder /usr/src/proyecto/api-rust/target/release/api-rust .

# Exponer puerto HTTP
EXPOSE 8080

CMD ["./api-rust"]