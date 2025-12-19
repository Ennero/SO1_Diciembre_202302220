package main

import (
    "context"
    "fmt"
    "log"
    "net"

    // Importamos el paquete generado (ajusta la ruta si tu go.mod se llama diferente)
    pb "grpc-server/pb/proto"
    "google.golang.org/grpc"
)

// server es la estructura que implementa la interfaz generada por gRPC
type server struct {
    pb.UnimplementedProductSaleServiceServer
}

// ProcesarVenta es la función que definimos en el .proto
func (s *server) ProcesarVenta(ctx context.Context, req *pb.ProductSaleRequest) (*pb.ProductSaleResponse, error) {
    // 1. Imprimir lo que recibimos (Simulación de procesamiento)
    fmt.Printf("💰 Venta Recibida -> Categoria: %v | ID: %s | Precio: %.2f | Cantidad: %d\n",
        req.GetCategoria(), req.GetProductoId(), req.GetPrecio(), req.GetCantidadVendida())

    // 2. Aquí más adelante agregaremos el código para enviar a Kafka
    
    // 3. Responder al cliente que todo salió bien
    return &pb.ProductSaleResponse{
        Estado: "Exito: Venta procesada por Deployment Go",
    }, nil
}

func main() {
    // Definir el puerto de escucha
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Fallo al escuchar en puerto 50051: %v", err)
    }

    // Crear el servidor gRPC
    s := grpc.NewServer()
    
    // Registrar nuestro servicio en el servidor
    pb.RegisterProductSaleServiceServer(s, &server{})

    log.Printf("🚀 Servidor gRPC escuchando en puerto 50051...")
    
    // Iniciar el servidor
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Fallo al iniciar el servidor: %v", err)
    }
}