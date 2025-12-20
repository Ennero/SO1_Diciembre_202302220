# API REST en Rust (Frontend / Entry Point)

Este componente actúa como la puerta de entrada del sistema. Expone una API REST de alto rendimiento utilizando **Actix-Web** para recibir peticiones HTTP (JSON) y las redirige al servidor de procesamiento en Go mediante **gRPC**.

## Funcionalidad
1. **API Gateway:** Escucha peticiones HTTP POST en el puerto **8080**.
2. **Conversión de Protocolos:** Recibe JSON y lo transforma en mensajes Protobuf (`ProductSaleRequest`).
3. **Cliente gRPC:** Se conecta al servicio de Go (puerto 50051) para enviar los datos de la venta.

## Tecnologías Clave
* **Lenguaje:** Rust (Edition 2021).
* **Web Framework:** Actix-Web.
* **gRPC Client:** Tonic.
* **Compilación:** `build.rs` compila automáticamente el archivo `../proto/ventas.proto` usando `protobuf-compiler` del sistema.

## Requisitos Previos
* Rust instalado (`cargo`).
* Compilador de Protobuf instalado en el sistema:
    ```bash
    sudo apt-get install -y protobuf-compiler libprotobuf-dev
    ```

## Instrucciones de Ejecución

1.  **Iniciar el servidor:**
    ```bash
    cargo run
    ```
    *La primera ejecución puede tardar unos minutos compilando dependencias.*

2.  **Prueba manual (cURL):**
    ```bash
    curl -X POST http://localhost:8080/venta \
      -H "Content-Type: application/json" \
      -d '{"categoria": 1, "producto_id": "TEST-01", "precio": 100.0, "cantidad_vendida": 1}'
    ```

## Estructura de Archivos
* `src/main.rs`: Lógica principal del servidor HTTP y cliente gRPC.
* `build.rs`: Script de pre-compilación para generar código Rust desde el `.proto`.
* `Cargo.toml`: Gestión de dependencias (Actix, Tonic, Tokio, Serde).