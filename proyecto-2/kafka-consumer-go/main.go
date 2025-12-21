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

type VentaMsg struct {
	Categoria       int     `json:"categoria"`
	ProductoID      string  `json:"producto_id"`
	Precio          float64 `json:"precio"`
	CantidadVendida int     `json:"cantidad_vendida"`
}

// Mapa de IDs a Nombres
func obtenerNombreCategoria(id int) string {
	switch id {
	case 1:
		return "Electronica"
	case 2:
		return "Ropa"
	case 3:
		return "Hogar"
	case 4:
		return "Belleza"
	default:
		return "Otros"
	}
}

func main() {
	// --- Configuración ---
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "my-cluster-kafka-bootstrap:9092"
	}
	redisAddr := "valkey-service:6379" // Nombre del servicio en K8s
	fmt.Printf("Consumidor v4 Iniciado\nKafka: %s\nValkey: %s\n", kafkaBroker, redisAddr)

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    "ventas",
		GroupID:  "consumidores-grupo-v4",
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  1 * time.Second,
	})
	defer r.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr, Password: "", DB: 0})
	ctx := context.Background()

	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("Error Kafka: %v", err)
			break
		}

		var v VentaMsg
		if err := json.Unmarshal(m.Value, &v); err != nil {
			continue
		}

		catNombre := obtenerNombreCategoria(v.Categoria)

		// ---------------------------------------------------------
		// 1. CÁLCULO DE PROMEDIOS (CANTIDAD Y PRECIO)
		// ---------------------------------------------------------
		
		rdb.HIncrByFloat(ctx, "aux:suma_precio:"+catNombre, "total", v.Precio)
		rdb.HIncrBy(ctx, "aux:suma_cantidad:"+catNombre, "total", int64(v.CantidadVendida))
		rdb.HIncrBy(ctx, "aux:conteo_tx:"+catNombre, "total", 1)

		// Leemos los valores actuales para calcular el promedio al vuelo
		sumaPrecio, _ := rdb.HGet(ctx, "aux:suma_precio:"+catNombre, "total").Float64()
		sumaCant, _ := rdb.HGet(ctx, "aux:suma_cantidad:"+catNombre, "total").Float64()
		conteo, _ := rdb.HGet(ctx, "aux:conteo_tx:"+catNombre, "total").Float64()

		if conteo > 0 {
			promPrecio := sumaPrecio / conteo
			promCant := sumaCant / conteo

			// Guardamos los resultados finales para Grafana
			// Fila 1: Promedio Cantidad
			rdb.HSet(ctx, "stats:promedio_cantidad", catNombre, promCant)
			// Fila 2: Promedio Precio
			rdb.HSet(ctx, "stats:promedio_precio", catNombre, promPrecio)
		}

		// ---------------------------------------------------------
		// 2. OTRAS ESTADÍSTICAS GENERALES
		// ---------------------------------------------------------

		// Total Reportes (Ventas) por Categoría
		rdb.HIncrBy(ctx, "stats:reportes_categoria", catNombre, 1)

		// Precios Máximos y Mínimos (Global)
		memberID := fmt.Sprintf("%s-%d", v.ProductoID, time.Now().UnixNano()) // ID único
		rdb.ZAdd(ctx, "stats:precios_global", redis.Z{Score: v.Precio, Member: memberID})

		// Top Productos Más Vendidos (Global)
		rdb.ZIncrBy(ctx, "stats:productos_top", float64(v.CantidadVendida), v.ProductoID)

		// ---------------------------------------------------------
		// 3. SECCIÓN ESPECÍFICA (ELECTRONICA - CARNET 0)
		// ---------------------------------------------------------
		if catNombre == "Electronica" {
			// Top Productos (Solo Electronica)
			rdb.ZIncrBy(ctx, "stats:electronica:productos", float64(v.CantidadVendida), v.ProductoID)

			// Historial de Precios (Stream para Time Series)
			// "stream:electronica:precio"
			rdb.XAdd(ctx, &redis.XAddArgs{
				Stream: "stream:electronica:precio",
				MaxLen: 1000, // Guardar solo los últimos 1000 para no llenar la memoria
				Values: map[string]interface{}{
					"precio": v.Precio,
					"producto": v.ProductoID,
				},
			})
		}

		fmt.Printf("Procesado: %s (Cant: %d, $%.2f) - %s\n", v.ProductoID, v.CantidadVendida, v.Precio, catNombre)
	}
}