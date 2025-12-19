# Correr el protoc

Para poder correr el protoc, se utilizó un contenedor para facilitar el proceso y no tener que instalar plugins complejos:

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