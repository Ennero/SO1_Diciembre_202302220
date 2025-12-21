package main

// Servidor gRPC en Go que recibe ventas y las envía a Kafka.
import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	pb "grpc-server/pb/proto"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

// Estructura para enviar a Kafka como JSON
type VentaMsg struct {
	Categoria       int32   `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int32   `json:"cantidad_vendida"`
	Fecha           string  `json:"fecha"`
}

// Implementación del servicio gRPC
type server struct {
	pb.UnimplementedProductSaleServiceServer
	kafkaWriter *kafka.Writer
}

// ProcesarVenta recibe una venta y la envía a Kafka
func (s *server) ProcesarVenta(ctx context.Context, req *pb.ProductSaleRequest) (*pb.ProductSaleResponse, error) {
	// Crear el objeto JSON
	venta := VentaMsg{
		Categoria: int32(req.Categoria),
		ProductoID:      req.ProductoId,
		Precio:          req.Precio,
		CantidadVendida: req.CantidadVendida,
		Fecha:           time.Now().Format(time.RFC3339),
	}

	// Convertir a bytes (JSON String)
	msgBytes, err := json.Marshal(venta)
	if err != nil {
		fmt.Printf("Error creando JSON: %v\n", err)
		return &pb.ProductSaleResponse{Estado: "Error interno JSON"}, nil
	}

	// Escribir en Kafka
	err = s.kafkaWriter.WriteMessages(ctx, kafka.Message{
		Key:   []byte(req.ProductoId), // Usamos el ID como llave para ordenamiento
		Value: msgBytes,
	})

	// Manejo de errores al enviar a Kafka
	if err != nil {
		fmt.Printf("Error enviando a Kafka: %v\n", err)
		return &pb.ProductSaleResponse{Estado: "Fallo al enviar a Kafka"}, nil
	}

	// Éxito
	fmt.Printf("Venta enviada a Kafka: %s\n", string(msgBytes))
	return &pb.ProductSaleResponse{Estado: "Exito: Venta guardada en Kafka"}, nil
}

func main() {
	// En Kubernetes, el host será "kafka-service:9092"
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092" // Valor por defecto para pruebas locales
	}
	kafkaTopic := "ventas"

	fmt.Printf("Conectando a Kafka en: %s (Topic: %s)\n", kafkaBroker, kafkaTopic)

	// Inicializar el escritor de Kafka
	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    kafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	// Iniciar servidor gRPC
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Fallo al escuchar: %v", err)
	}

	// Crear servidor gRPC
	s := grpc.NewServer()
	pb.RegisterProductSaleServiceServer(s, &server{kafkaWriter: writer})

	// Iniciar servicio
	log.Printf("Servidor gRPC escuchando en puerto :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Fallo al servir: %v", err)
	}
}