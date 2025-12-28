use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};
use reqwest::Client;

// Estructura input (Locust) y output (hacia Go Client)
#[derive(Deserialize, Serialize)]
struct VentaInput {
    categoria: i32,
    producto_id: String,
    precio: f64,
    cantidad_vendida: i32,
}

#[post("/venta")]
async fn registrar_venta(item: web::Json<VentaInput>, client: web::Data<Client>) -> impl Responder {
    let url = "http://go-http-service:3000/venta";

    let response = client.post(url)
        .json(&item)
        .send()
        .await;

    // Manejo de la respuesta
    match response {
        Ok(resp) => {
            if resp.status().is_success() {
                let body = resp.text().await.unwrap_or("OK".to_string());
                HttpResponse::Ok().body(format!("Go Client respondió: {}", body))
            } else {
                HttpResponse::InternalServerError().body("Error en Go Client")
            }
        },
        Err(e) => HttpResponse::InternalServerError().body(format!("Fallo HTTP a Go: {}", e)),
    }
}

// Servidor Actix-web
#[actix_web::main]
async fn main() -> std::io::Result<()> {
    println!("API Rust (HTTP Client) escuchando en 8080");
    let client = Client::new();
    let bind_addr = std::env::var("BIND_ADDR")
        .map_err(|_| std::io::Error::new(std::io::ErrorKind::InvalidInput, "Configura BIND_ADDR"))?;

    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(client.clone()))
            .service(registrar_venta)
    })
    .bind((bind_addr.as_str(), 8080))?
    .run()
    .await
}