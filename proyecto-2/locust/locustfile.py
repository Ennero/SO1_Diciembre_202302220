from locust import HttpUser, task, between
import random

class TraficoVentas(HttpUser):
    wait_time = between(0.1, 0.5)

    @task
    def enviar_venta(self):
        # Productos variados
        productos_electronica = ["Mouse-Gamer", "Teclado-RGB", "Monitor-4K", "GPU-4090", "Arduino"]
        productos_ropa = ["Camiseta-Geek", "Hoodie-Python", "Gorra-Linux"]
        productos_hogar = ["Lampara-LED", "Silla-Ergo"]
        
        categoria_id = random.randint(1, 4) # 1=Electronica, 2=Ropa, 3=Hogar, 4=Belleza
        
        producto = "Generico"
        if categoria_id == 1:
            producto = random.choice(productos_electronica)
        elif categoria_id == 2:
            producto = random.choice(productos_ropa)
        else:
            producto = random.choice(productos_hogar)

        payload = {
            "categoria": categoria_id,
            "producto_id": producto,
            "precio": round(random.uniform(10.0, 500.0), 2),
            "cantidad_vendida": random.randint(1, 5)
        }
        self.client.post("/venta", json=payload)