use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::Deserialize;
use tonic::Request;

// Importar el código generado por build.rs
pub mod blackfriday {
    tonic::include_proto!("blackfriday");
}
use blackfriday::product_sale_service_client::ProductSaleServiceClient;
use blackfriday::{ProductSaleRequest, CategoriaProducto};

// Estructura para recibir el JSON de Locust
#[derive(Deserialize)]
struct VentaInput {
    categoria: i32, // Recibimos 1, 2, 3... igual que el enum
    producto_id: String,
    precio: f64,
    cantidad_vendida: i32,
}

#[post("/venta")]
async fn registrar_venta(item: web::Json<VentaInput>) -> impl Responder {
    // 1. Conectar al servidor gRPC (Go)
    // Nota: En un entorno real, la conexión se debería reusar, no crear por cada petición.
    let mut client = match ProductSaleServiceClient::connect("http://[::1]:50051").await {
        Ok(c) => c,
        Err(e) => return HttpResponse::InternalServerError().body(format!("Error conectando gRPC: {}", e)),
    };

    // 2. Crear el mensaje gRPC
    let request = Request::new(ProductSaleRequest {
        categoria: item.categoria, // Rust hará el cast automático si coincide
        producto_id: item.producto_id.clone(),
        precio: item.precio,
        cantidad_vendida: item.cantidad_vendida,
    });

    // 3. Enviar a Go
    match client.procesar_venta(request).await {
        Ok(response) => {
            let respuesta = response.into_inner();
            HttpResponse::Ok().body(format!("Go respondió: {}", respuesta.estado))
        }
        Err(e) => {
            HttpResponse::InternalServerError().body(format!("Fallo gRPC: {}", e))
        }
    }
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    println!("🦀 API Rust escuchando en puerto 8080");
    println!("➡️  Redirigiendo tráfico a gRPC puerto 50051");

    HttpServer::new(|| {
        App::new()
            .service(registrar_venta)
    })
    .bind(("0.0.0.0", 8080))?
    .run()
    .await
}