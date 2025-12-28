# Protocolos gRPC

Contratos compartidos en `ventas.proto`. Mantén este archivo como fuente única de verdad para clientes y servidores.

## Mensajes y servicio
- `ProductSaleRequest`: `categoria` (enum), `producto_id` (string), `precio` (double), `cantidad_vendida` (int32).
- `ProductSaleResponse`: `estado` (string).
- `CategoriaProducto`: UNKNOWN=0, ELECTRONICA=1, ROPA=2, HOGAR=3, BELLEZA=4.
- Servicio `ProductSaleService` con RPC `ProcesarVenta`.

## Generar código Go (desde la raíz `proyecto-2/`)
```bash
docker run --rm -v "$(pwd):/src" -w /src golang:1.23 /bin/bash -c " \
  apt-get update && apt-get install -y protobuf-compiler && \
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1 && \
  export PATH=\$PATH:\$(go env GOPATH)/bin && \
  mkdir -p grpc-server-go/pb && \
  protoc --go_out=./grpc-server-go/pb --go_opt=paths=source_relative \
         --go-grpc_out=./grpc-server-go/pb --go-grpc_opt=paths=source_relative \
         --proto_path=. proto/ventas.proto"
```

## Buenas prácticas
- No incluyas IPs o rutas específicas en los .proto ni en los comandos; parametriza con variables o paths relativos.
- Versiona el archivo .proto y regenera artefactos en los servicios que lo consumen.



