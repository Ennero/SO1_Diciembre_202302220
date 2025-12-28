# Configurar Grafana (11 paneles)

Instrucciones rápidas para recrear el dashboard conectado al datasource Redis. Reemplaza cualquier identificador personal por uno neutro (ej.: `<identificador de categoria>`).

## Prerrequisitos
- Data source: Redis (plugin `redis-datasource`).
- Modo de consulta: Command.

## Fila 1 – Visión general
1. **Bar Gauge** – Promedio de precio por categoría
	- Command: `HGETALL`
	- Key: `stats:promedio_precio`
2. **Stat** – Precio más alto (global)
	- Command: `ZREVRANGE`
	- Key: `stats:precios_global`
	- Min/Max: `0/0`, withscores ON
3. **Stat** – Producto más vendido (global)
	- Command: `ZREVRANGE`
	- Key: `stats:productos_top`
	- Min/Max: `0/0`

## Fila 2 – Distribución
4. **Bar Gauge** – Promedio de precio por categoría (mismo comando del panel 1)
5. **Stat** – Precio más bajo (global)
	- Command: `ZRANGE`
	- Key: `stats:precios_global`
	- Min/Max: `0/0`, withscores ON
6. **Stat** – Producto menos vendido (global)
	- Command: `ZRANGE`
	- Key: `stats:productos_top`
	- Min/Max: `0/0`

## Fila 3 – Conteos
7. **Bar Gauge** – Total de reportes por categoría
	- Command: `HGETALL`
	- Key: `stats:reportes_categoria`

## Fila 4 – Categoría destacada (ej.: electrónica)
8. **Text** – Identificador de categoría (usa `<identificador de categoria>`, tamaño grande)
9. **Text** – Nombre de categoría (ej.: `ELECTRONICA`)
10. **Stat** – Producto más vendido de la categoría
	 - Command: `ZREVRANGE`
	 - Key: `stats:electronica:productos`
	 - Min/Max: `0/0`
11. **Stat** – Producto menos vendido de la categoría
	 - Command: `ZRANGE`
	 - Key: `stats:electronica:productos`
	 - Min/Max: `0/0`
12. **Time series** – Variación de precio de la categoría
	 - Command: `XRANGE`
	 - Key: `stream:electronica:precio`
	 - Start/End: `- / +`

Si el plugin no soporta series de tiempo, usa un panel de tabla con los últimos valores del stream.