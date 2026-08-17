# Despliegue en Railway

## Topología recomendada

Mantener el servicio de precios separado del Gestor de Alimentación aunque compartan proyecto de Railway:

```text
prices-postgres
      ↑
      ├── prices-api
      ├── prices-migrate
      └── prices-worker
```

Los tres servicios de aplicación pueden desplegarse desde este mismo repositorio/imagen y compartir exclusivamente la base `prices-postgres`.

## Variables

Los servicios que acceden a la base necesitan:

```text
DATABASE_URL=<cadena de conexión de prices-postgres>
```

`prices-api` además recibe `PORT` de la plataforma.

No apuntar este repositorio a la base de datos del Gestor de Alimentación.

## Comandos de ejecución

### API

```text
prices-api
```

La API comprueba `DATABASE_URL` al arrancar. Si la variable existe y PostgreSQL no responde, el proceso falla en lugar de usar datos demo.

### Migraciones

```text
prices-migrate
```

El binario:

1. toma un advisory lock de PostgreSQL para evitar dos migradores simultáneos;
2. crea `schema_migrations` si es necesario;
3. aplica los `.sql` embebidos en orden alfabético;
4. registra cada migración aplicada;
5. omite migraciones ya registradas.

Las migraciones deben ejecutarse antes del primer import y antes de desplegar código que dependa de una migración nueva.

### Worker / importación DIA

Primera validación sin escribir:

```text
import-prices --supermarket=dia --postal-code=28001 --dry-run=true
```

Primera importación persistente controlada:

```text
import-prices --supermarket=dia --postal-code=28001 --dry-run=false
```

La configuración actual usa una única categoría de arroz por defecto. No ampliar a todo el catálogo hasta validar los resultados de esta primera importación.

## Secuencia de primera puesta en marcha

```text
1. Crear prices-postgres
2. Configurar DATABASE_URL en prices-migrate
3. Ejecutar prices-migrate
4. Ejecutar import-prices en dry-run
5. Revisar productos normalizados
6. Ejecutar import-prices con dry-run=false
7. Configurar DATABASE_URL en prices-api
8. Arrancar prices-api
9. Probar /api/v1/products/search?q=arroz&postalCode=28001
```

## Comprobaciones SQL útiles

Después de la primera importación:

```sql
SELECT supermarket_id, COUNT(*)
FROM supermarket_products
GROUP BY supermarket_id;
```

```sql
SELECT
    sp.supermarket_id,
    sp.external_id,
    sp.name,
    po.price,
    po.price_per_unit,
    po.postal_code,
    po.observed_at
FROM supermarket_products sp
JOIN price_observations po ON po.supermarket_product_id = sp.id
WHERE sp.supermarket_id = 'dia'
ORDER BY po.observed_at DESC
LIMIT 20;
```

```sql
SELECT COUNT(*) FROM price_promotions;
```

## Regla operativa

El frontend y `Gestor-Alimentacion` nunca deben llamar directamente a DIA, Mercadona o Lidl. Solo consumen `prices-api`. La adquisición de precios ocurre fuera del ciclo de petición del usuario.
