FROM golang:1.23 AS builder
WORKDIR /app

COPY . .


RUN go mod tidy

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o cliente-go main.go

FROM gcr.io/distroless/static-debian12
WORKDIR /app
COPY --from=builder /app/cliente-go .
EXPOSE 3000
CMD ["./cliente-go"]