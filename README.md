# Supermarket-Prices-API

API independiente para recopilar, normalizar, almacenar y comparar precios de supermercados en España.

## Objetivo del MVP

Supermercados prioritarios:

1. DIA
2. Mercadona
3. Lidl

El servicio mantiene separado el dominio de precios del `Gestor-Alimentacion`. El gestor consumirá esta API, pero no conocerá cómo se obtiene el precio de cada cadena.

## Arquitectura

```text
Supermercados
     ↓
prices-worker
     ↓
normalización
     ↓
PostgreSQL
     ↓
prices-api
     ↓
Gestor-Alimentacion / otros consumidores
```

Los extractores se implementan mediante providers independientes. La API pública consulta nuestro catálogo persistido y no realiza scraping durante una petición de usuario.

## Estado

### Fase 0 — completada

- Go + API HTTP.
- Modelo de catálogo normalizado.
- Interfaz de supermercado `Provider`.
- PostgreSQL y migraciones iniciales.
- Docker/Railway.
- CI con tests y `go vet`.
- Documentación de arquitectura.

### Fase 1 — DIA en desarrollo

- Normalización de productos/precios/promociones.
- Parser reproducible de listados.
- `HTTPSource` para páginas públicas de categoría.
- Importer desacoplado mediante `Provider -> Sink`.
- Modo dry-run.
- `PostgresSink` para guardar producto, observaciones históricas y promociones.
- `CatalogStore` PostgreSQL para servir el último precio relevante por código postal.
- La API usa PostgreSQL cuando `DATABASE_URL` está configurada y solo usa datos demo en desarrollo sin base configurada.

## Ejecutar API

Sin `DATABASE_URL`, la API arranca con catálogo demo local:

```bash
go run ./cmd/api
```

Con PostgreSQL, las búsquedas se resuelven contra los productos importados:

```bash
DATABASE_URL='postgres://...' go run ./cmd/api
```

Si `DATABASE_URL` está presente pero la conexión falla, el proceso no arranca. Si una lectura del catálogo falla durante una petición, la API devuelve `500 catalog_unavailable`; no sustituye silenciosamente la base real por datos demo.

Endpoints iniciales:

```text
GET /health
GET /api/v1/version
GET /api/v1/supermarkets
GET /api/v1/products/search?q=arroz&postalCode=28001
```

La búsqueda devuelve como máximo 100 productos y, para cada producto, selecciona la observación más reciente compatible con el código postal solicitado, dando prioridad a una observación específica de ese código postal frente a una observación global.

## Importer DIA

Inspeccionar una importación sin escribir en la base:

```bash
go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --dry-run=true
```

Persistir en PostgreSQL (requiere migraciones aplicadas y `DATABASE_URL`):

```bash
go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --dry-run=false
```

Más detalles en `docs/DIA_IMPORTER.md`.

## Tests

```bash
go test ./...
go vet ./...
```

## Roadmap

- Fase 1A: DIA.
- Fase 1B: Mercadona.
- Fase 1C: Lidl.
- Fase 2: catálogo canónico y matching entre cadenas.
- Fase 3: comparación de cesta en uno o varios supermercados.
- Fase 4: preferencias de tienda, marca, calidad y fidelización.
- Fase 5: integración con `Gestor-Alimentacion`.

La planificación detallada se mantiene en los issues del repositorio.
