# Configurar Grafana (Las 11 Gráficas) 📊

Abre Grafana (http://TU_IP:80), crea un Dashboard Nuevo y agrega 11 paneles.

**Nota importante:** En Grafana, selecciona el Data Source **Redis** y usa el modo **"Command"** para escribir los comandos tal cual.

---

## FILA 1: Estadísticas Generales

### 1. Gráfica Barras: Producto Promedio por Categoría
(Entiendo esto como promedio de precio, ya que la cantidad es rara promediarla).

- **Visualization:** Bar Gauge
- **Command:** `HGETALL`
- **Key:** `stats:promedio_precio`
- **Type:** HGETALL (Esto te mostrará una barra por categoría: Electronica, Ropa, etc.)

### 2. Stat: Precio Más Alto (General)

- **Visualization:** Stat
- **Command:** `ZREVRANGE` (Trae los valores más altos primero)
- **Key:** `stats:precios_global`
- **Min:** 0 **Max:** 0 (Trae el top 1)
- **Withscores:** Actívalo (para ver el precio)

### 3. Stat: Producto Más Vendido (General)

- **Visualization:** Stat
- **Command:** `ZREVRANGE`
- **Key:** `stats:productos_top`
- **Min:** 0 **Max:** 0
- (Mostrará el ID del producto que más se vendió)

---

## FILA 2

### 4. Gráfica Barras: Precio Promedio por Categoría

**Nota:** La gráfica 1 y 4 son redundantes en tu imagen, usa el mismo comando de la 1 o cambia una a "Total ventas". Usemos el mismo:

- **Command:** `HGETALL`
- **Key:** `stats:promedio_precio`

### 5. Stat: Precio Más Bajo (General)

- **Visualization:** Stat
- **Command:** `ZRANGE` (Trae los valores más bajos primero)
- **Key:** `stats:precios_global`
- **Min:** 0 **Max:** 0
- **Withscores:** ON

### 6. Stat: Producto Menos Vendido (General)

- **Visualization:** Stat
- **Command:** `ZRANGE`
- **Key:** `stats:productos_top`
- **Min:** 0 **Max:** 0

---

## FILA 3

### 7. Gráfica Barras: Total de Reportes por Categoría

- **Visualization:** Bar Chart (o Bar Gauge)
- **Command:** `HGETALL`
- **Key:** `stats:reportes_categoria`

---

## FILA 4: SECCIÓN ELECTRÓNICA (Tu Carnet)

### 8. Texto: #CARNET

- **Visualization:** Text
- **Content:** `202302220` (Tamaño gigante)

### 9. Texto: NOMBRE CATEGORÍA

- **Visualization:** Text
- **Content:** `ELECTRONICA` (Color azul)

### 10. Stat: Producto Más Vendido (Electronica)

- **Command:** `ZREVRANGE`
- **Key:** `stats:electronica:productos`
- **Min:** 0 **Max:** 0

### 11. Stat: Producto Menos Vendido (Electronica)

- **Command:** `ZRANGE`
- **Key:** `stats:electronica:productos`
- **Min:** 0 **Max:** 0

### 12. Time Series: Variación de Precio (Electronica)

- **Visualization:** Time Series
- **Type:** RedisGears o Streams (depende del plugin), pero lo más fácil en Grafana básico con Redis es:
- **Command:** `XRANGE`
- **Key:** `stream:electronica:precio`
- **Start:** `-` **End:** `+`

**Nota:** Si el gráfico de serie de tiempo es difícil de configurar con Redis básico, usa una tabla (Table) con los últimos valores del stream.