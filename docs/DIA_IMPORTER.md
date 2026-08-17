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
Sink (dry-run ahora; PostgreSQL a continuación)
```

## Dry-run

```bash
go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --dry-run
```

Por defecto se usa una única categoría pequeña de arroz para validar el pipeline sin recorrer todavía todo el catálogo.

Se pueden proporcionar categorías explícitas:

```bash
go run ./cmd/import-prices \
  --supermarket=dia \
  --postal-code=28001 \
  --categories='https://www.dia.es/arroz-pastas-y-legumbres/arroz/c/L2042?page=1' \
  --dry-run
```

El comando realiza una sola descarga por URL en cada ejecución y devuelve JSON con el resumen y los productos normalizados.

## Persistencia

El paquete `internal/importer` define `Sink.SaveProducts`. El siguiente paso es implementar un `PostgresSink` que haga upsert de `supermarket_products` e inserte observaciones inmutables en `price_observations`/`product_promotions`.

Mantener esta frontera permite probar crawler y normalización sin PostgreSQL y probar persistencia sin red externa.

## Reglas de adquisición

- Usar páginas públicas de categoría/listado, no rutas de búsqueda bloqueadas.
- User-Agent identificable.
- Timeouts y límites de tamaño de respuesta.
- Nada de scraping desde el frontend.
- No ejecutar una petición a DIA por cada búsqueda del usuario.
- Tests sin acceso a red.

## Alcance actual

Esta etapa prueba la arquitectura y un conjunto de categorías controlado. Antes de ampliar a todo el catálogo se debe validar el HTML real de varias familias de productos, paginación, ubicación, promociones y productos agotados.
