# API Rust (puerta de entrada HTTP)

Servicio Actix que recibe peticiones JSON y las reenvía vía HTTP al cliente Go (`go-http-service`). No mantiene lógica de negocio; solo valida y reenvía.

## Endpoints
- `POST /venta`: recibe `{categoria, producto_id, precio, cantidad_vendida}` y devuelve el cuerpo que responde el cliente Go.

## Variables de entorno
- `BIND_ADDR`: dirección de escucha (ej.: `<direccion IP de escucha>`). Debe ser provista por el entorno; no se versiona ninguna IP.
- `GO_CLIENT_URL` (opcional): URL del servicio intermedio. Por defecto `http://go-http-service:3000/venta` en Kubernetes.

## Ejecutar en local
```bash
cargo run
curl -X POST http://localhost:8080/venta \
    -H "Content-Type: application/json" \
    -d '{"categoria": 1, "producto_id": "TEST-01", "precio": 100.0, "cantidad_vendida": 1}'
```

## Docker (ejemplo de build)
```bash
docker build -f api-rust/Dockerfile -t <direccion IP del registry privado>:5000/api-rust:v3 .
docker push <direccion IP del registry privado>:5000/api-rust:v3
```

## Archivos principales
- `src/main.rs`: controlador Actix y reenvío HTTP.
- `Cargo.toml`: dependencias (actix-web, reqwest, serde, tokio).
- `Dockerfile`: build multi-etapa y ejecución en Debian slim.