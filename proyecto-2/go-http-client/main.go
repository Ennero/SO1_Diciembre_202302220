package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	pb "go-http-client/pb/proto"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Estructura para recibir el JSON de Rust
type VentaInput struct {
	Categoria       int32   `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int32   `json:"cantidad_vendida"`
}

func main() {
	// 1. Configurar conexión gRPC hacia el Backend (Deployment 2)
	backendHost := os.Getenv("GRPC_SERVER_HOST")
	if backendHost == "" {
		backendHost = "grpc-go-service:50051"
	}

	fmt.Printf("Conectando gRPC a: %s\n", backendHost)
	conn, err := grpc.Dial(backendHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("No se pudo conectar al backend gRPC: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewProductSaleServiceClient(conn)

	// 2. Configurar Servidor HTTP (Fiber)
	app := fiber.New()

	app.Post("/venta", func(c *fiber.Ctx) error {
		var input VentaInput
		if err := c.BodyParser(&input); err != nil {
			return c.Status(400).SendString("JSON inválido")
		}

		// 3. Convertir y enviar vía gRPC
		req := &pb.ProductSaleRequest{
			Categoria:       pb.CategoriaProducto(input.Categoria),
			ProductoId:      input.ProductoID,
			Precio:          input.Precio,
			CantidadVendida: input.CantidadVendida,
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		resp, err := grpcClient.ProcesarVenta(ctx, req)
		if err != nil {
			return c.Status(500).SendString(fmt.Sprintf("Error gRPC: %v", err))
		}

		return c.Status(200).SendString(resp.Estado)
	})

	log.Println("Go HTTP Client escuchando en puerto 3000")
	log.Fatal(app.Listen(":3000"))
}