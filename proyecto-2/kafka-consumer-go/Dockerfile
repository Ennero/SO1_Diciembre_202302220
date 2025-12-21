# Compilación
FROM golang:1.23 as builder
WORKDIR /app

# Copiar dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar (CGO_ENABLED=0 para que sea estático y ligero)
RUN CGO_ENABLED=0 GOOS=linux go build -o consumidor main.go

# Imagen final ligera
FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/consumidor .
CMD ["./consumidor"]