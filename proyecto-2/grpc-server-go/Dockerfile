# Etapa 1: Compilación
FROM golang:1.23 AS builder
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar el código fuente (incluyendo la carpeta pb generada)
COPY . .

# Compilar el binario
RUN CGO_ENABLED=0 GOOS=linux go build -o servidor-go main.go

# Etapa 2: Imagen final ligera
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/servidor-go .

# Exponer puerto gRPC
EXPOSE 50051

# Ejecutar
CMD ["./servidor-go"]