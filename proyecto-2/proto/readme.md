# Definición de Protocolos (gRPC)

Esta carpeta contiene los archivos `.proto` que definen los contratos de comunicación entre los microservicios del sistema (Rust y Go).

## Archivo: ventas.proto
Define la estructura de los mensajes de ventas del Black Friday y el servicio RPC disponible.

### Estructura del Mensaje
- **ProductSaleRequest:**
  - `categoria` (Enum)
  - `producto_id` (String)
  - `precio` (Double)
  - `cantidad_vendida` (Int32)

### Generación de Código

Para generar los archivos `.pb.go` necesarios para el servidor en Go, utilizamos Docker para evitar instalar dependencias locales.

**Comando de Generación (Ejecutar desde la raíz `proyecto-2/`):**

```bash
sudo docker run --rm -v "$(pwd):/src" -w /src golang:1.23 /bin/bash -c " \
  apt-get update && apt-get install -y protobuf-compiler && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1 && \
  export PATH=\$PATH:\$(go env GOPATH)/bin && \
  mkdir -p grpc-server-go/pb && \
  protoc --go_out=./grpc-server-go/pb --go_opt=paths=source_relative \
         --go-grpc_out=./grpc-server-go/pb --go-grpc_opt=paths=source_relative \
         --proto_path=. proto/ventas.proto"
```



