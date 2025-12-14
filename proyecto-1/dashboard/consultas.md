# CÓDIGO SQL PARA GRAFANA (16 Gráficos)

A continuación, se presentan las consultas de sql para graficar en grafana.

---

## A. DASHBOARD 1: CONTENEDORES (Filtrado)

Este dashboard se enfoca solo en `stress-ng` y `sleep`.

### 1. Total RAM (Stat)
```sql
SELECT
  total as "Total RAM"
FROM ram_log
ORDER BY id DESC LIMIT 1;
```

### 2. Free RAM (Stat)
```sql
SELECT
  (total - used) as "Free RAM"
FROM ram_log
ORDER BY id DESC LIMIT 1;
```

### 3. Contenedores Eliminados (Time Series o Bar Chart)

Muestra cuándo ocurrieron las eliminaciones.
```sql
SELECT
  timestamp as time,
  count(id) as value
FROM kill_log
WHERE $__timeFilter(timestamp)
GROUP BY timestamp
ORDER BY timestamp ASC;
```

### 4. Gráfica de Uso de RAM en el tiempo (Time Series)
```sql
SELECT
  timestamp as time,
  used as "RAM Usada (MB)"
FROM ram_log
WHERE $__timeFilter(timestamp)
ORDER BY timestamp ASC;
```

### 5. Top 5 Contenedores +RAM (Pie Chart)

Filtramos por nombre. Usamos `MAX` para ver el pico alcanzado aunque ya se haya eliminado.
```sql
SELECT
  name || ' (' || pid || ')' as metric,
  MAX(ram) as value
FROM process_log
WHERE $__timeFilter(timestamp) 
  AND (name LIKE 'stress-ng%' OR name = 'sleep')
GROUP BY pid, name
ORDER BY value DESC
LIMIT 5;
```

### 6. Top 5 Contenedores +CPU (Pie Chart)
```sql
SELECT
  name || ' (' || pid || ')' as metric,
  MAX(cpu) as value
FROM process_log
WHERE $__timeFilter(timestamp) 
  AND (name LIKE 'stress-ng%' OR name = 'sleep')
GROUP BY pid, name
ORDER BY value DESC
LIMIT 5;
```

### 7. RAM Usada (Stat)
```sql
SELECT
  used as "RAM Usada"
FROM ram_log
ORDER BY id DESC LIMIT 1;
```

### 8. Gráfico Propuesto: Conteo de Contenedores Activos (Time Series)

Cuenta cuántos procesos de tus contenedores estaban vivos en cada momento.
```sql
SELECT
  timestamp as time,
  count(distinct pid) as "Contenedores Vivos"
FROM process_log
WHERE $__timeFilter(timestamp) 
  AND (name LIKE 'stress-ng%' OR name = 'sleep')
GROUP BY timestamp
ORDER BY timestamp ASC;
```

---

## B. DASHBOARD 2: SISTEMA (General)

Este dashboard muestra todos los procesos, tal como lo pide la imagen.

### 1. Total RAM (Stat)

(Igual que el anterior)
```sql
SELECT total as "Total RAM" FROM ram_log ORDER BY id DESC LIMIT 1;
```

### 2. Free RAM (Stat)

(Igual que el anterior)
```sql
SELECT (total - used) as "Free RAM" FROM ram_log ORDER BY id DESC LIMIT 1;
```

### 3. Total Procesos Contados (Stat o Gauge)

Cuenta cuántos procesos únicos se detectaron en el último escaneo.
```sql
SELECT
  count(distinct pid)
FROM process_log
WHERE timestamp = (SELECT MAX(timestamp) FROM process_log);
```

### 4. Gráfica de Uso de RAM en el tiempo (Time Series)

(Igual que el anterior, la RAM global es la misma)
```sql
SELECT
  timestamp as time,
  used as "RAM Usada (MB)"
FROM ram_log
WHERE $__timeFilter(timestamp)
ORDER BY timestamp ASC;
```

### 5. Top 5 Procesos del Sistema +RAM (Pie Chart)

**NOTA:** Aquí NO filtramos por nombre. Verás procesos como `gnome-shell`, `Xorg`, o tu propio `daemon`.
```sql
SELECT
  name || ' (' || pid || ')' as metric,
  MAX(ram) as value
FROM process_log
WHERE $__timeFilter(timestamp)
GROUP BY pid, name
ORDER BY value DESC
LIMIT 5;
```

### 6. Top 5 Procesos del Sistema +CPU (Pie Chart)
```sql
SELECT
  name || ' (' || pid || ')' as metric,
  MAX(cpu) as value
FROM process_log
WHERE $__timeFilter(timestamp)
GROUP BY pid, name
ORDER BY value DESC
LIMIT 5;
```

### 7. RAM Usada (Stat)

(Igual que el anterior)
```sql
SELECT used as "RAM Usada" FROM ram_log ORDER BY id DESC LIMIT 1;
```

### 8. Gráfico Propuesto: Carga promedio de CPU del Sistema (Time Series)

Promedio de CPU de todos los procesos en cada instante.
```sql
SELECT
  timestamp as time,
  avg(cpu) as "CPU Promedio Sistema"
FROM process_log
WHERE $__timeFilter(timestamp)
GROUP BY timestamp
ORDER BY timestamp ASC;
```