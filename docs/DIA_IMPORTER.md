# DIA importer

## Objetivo

El importer descarga páginas públicas de categoría/listado de DIA, extrae productos, normaliza los datos y los entrega a un `Sink` de persistencia.

La API pública no consulta DIA en tiempo real. Las búsquedas de usuarios se resolverán contra nuestro catálogo persistido.

## Flujo

```text
DIA category pages
      ↓
HTTPSource
      ↓
RawProduct
      ↓
Provider / Normalize
      ↓
Importer
      ↓
Memory sink (dry-run) o PostgreSQL
```

## Dry-run

```bash
go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --dry-run=true
```

Por defecto se usa una única categoría pequeña de arroz para validar el pipeline sin recorrer todavía todo el catálogo.

Se pueden proporcionar categorías explícitas:

```bash
go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --categories='https://www.dia.es/arroz-pastas-y-legumbres/arroz/c/L2042?page=1' \
  --dry-run=true
```

El comando realiza una sola descarga por URL en cada ejecución y devuelve JSON con el resumen y los productos normalizados.

## Persistencia PostgreSQL

Para persistir, las migraciones `001_init.sql` y `002_promotions_and_mvp.sql` deben estar aplicadas y `DATABASE_URL` debe apuntar a la base de datos del servicio de precios.

```bash
export DATABASE_URL='postgres://...'

go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --dry-run=false
```

`PostgresSink` ejecuta el lote dentro de una transacción:

1. Hace `UPSERT` de `supermarket_products` usando `(supermarket_id, external_id)`.
2. Inserta siempre una fila nueva en `price_observations`.
3. Inserta las promociones de esa observación en `price_promotions`.
4. Si falla cualquier producto del lote, la transacción completa se revierte.

De esta forma los metadatos del producto pueden evolucionar sin destruir el histórico de precios.

## Reglas de adquisición

- Usar páginas públicas de categoría/listado, no rutas de búsqueda bloqueadas.
- User-Agent identificable.
- Timeouts y límites de tamaño de respuesta.
- Nada de scraping desde el frontend.
- No ejecutar una petición a DIA por cada búsqueda del usuario.
- Tests sin acceso a red.

## Alcance actual

Esta etapa prueba la arquitectura y un conjunto de categorías controlado. Antes de ampliar a todo el catálogo se debe validar el HTML real de varias familias de productos, paginación, ubicación, promociones y productos agotados.

## Siguiente paso

- Validar el modo persistente contra un PostgreSQL real de desarrollo/Railway.
- Añadir tests de integración de persistencia.
- Añadir paginación/cobertura de categorías y métricas de importación.
- Sustituir el catálogo en memoria de la API por consultas PostgreSQL.
