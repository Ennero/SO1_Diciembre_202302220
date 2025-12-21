package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// Estructura de la venta (la misma que envía el server)
type VentaMsg struct {
	ProductoID string `json:"producto_id"`
}

func main() {
	// --- Configuración ---
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	kafkaTopic := "ventas"
	kafkaGroupID := "consumidores-grupo-1"
	redisAddr := "valkey-service:6379" // Dirección del servicio de la VM

	if kafkaBroker == "" {
		kafkaBroker = "my-cluster-kafka-bootstrap:9092"
	}

	fmt.Printf("📥 Consumidor v2 Iniciado\nKafka: %s\nValkey VM: %s\n", kafkaBroker, redisAddr)

	// --- Conexión a Kafka ---
	// Usamos NewReader con configuración explícita para evitar perder mensajes
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     kafkaTopic,
		GroupID:   kafkaGroupID,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
		MaxWait:   1 * time.Second,
		StartOffset: kafka.FirstOffset, // Importante: Leer desde el inicio si es nuevo
	})
	defer r.Close()

	// --- Conexión a Valkey (Redis) ---
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // Sin contraseña por defecto
		DB:       0,  // DB por defecto
	})

	ctx := context.Background()
	fmt.Println("🎧 Escuchando ventas y guardando en VM...")

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("❌ Error leyendo Kafka: %v", err)
			break
		}

		// 1. Desempaquetar JSON
		var venta VentaMsg
		if err := json.Unmarshal(m.Value, &venta); err != nil {
			log.Printf("⚠️ Error entendiendo JSON: %v", err)
			continue
		}

		// 2. Guardar en Valkey (Incrementar contador)
		// Comando Redis: INCR "contador:Monitor-4K"
		key := fmt.Sprintf("contador:%s", venta.ProductoID)
		err = rdb.Incr(ctx, key).Err()

		if err != nil {
			// ESTO FALLARÁ HASTA ARREGLAR LA VM (Es esperado por ahora)
			fmt.Printf("❌ Error conectando a Valkey: %v (Venta recibida: %s)\n", err, venta.ProductoID)
		} else {
			fmt.Printf("💾 Guardado en DB: %s (+1)\n", venta.ProductoID)
		}
	}
}