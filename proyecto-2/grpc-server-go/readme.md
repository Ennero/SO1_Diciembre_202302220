# Servidor gRPC en Go (Writer & Processor)

Este microservicio actúa como servidor gRPC encargado de recibir las transacciones de ventas provenientes de la API (Rust) o del Cliente gRPC.

## Funcionalidad
1. Escucha peticiones en el puerto **50051**.
2. Implementa el servicio `ProductSaleService` definido en `proto/ventas.proto`.
3. Procesa la venta e imprime los detalles en consola (Fase actual).
4. (Próximamente) Publicará los mensajes en un tópico de Kafka.

## Estructura del Proyecto

- **`main.go`**: Punto de entrada del servidor. Contiene la lógica del método `ProcesarVenta`.
- **`pb/`**: Contiene el código autogenerado por Protocol Buffers (`ventas.pb.go` y `ventas_grpc.pb.go`). **No editar manualmente**.
- **`go.mod` / `go.sum`**: Gestión de dependencias de Go.

## Requisitos Previos
- Go 1.23 o superior.
- Haber generado los archivos `.pb.go` (ver README de la carpeta `../proto`).

## Instrucciones de Ejecución

1. **Instalar dependencias:**
   ```bash
   go mod tidy
   ```


## Iniciar el servidor

```bash
go run main.go
```

## Verificación: Deberías ver el mensaje:

```bash
🚀 Servidor gRPC escuchando en puerto 50051...
```
