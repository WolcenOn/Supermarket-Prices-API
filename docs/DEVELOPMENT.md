# Desarrollo

## Requisitos

- Go 1.24+
- PostgreSQL para las fases con persistencia real
- Docker opcional

## Ejecutar la API

```bash
go run ./cmd/api
```

Por defecto escucha en `:8080`.

## Comprobaciones

```bash
go test ./...
go vet ./...
```

Endpoints iniciales:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/version
curl http://localhost:8080/api/v1/supermarkets
curl 'http://localhost:8080/api/v1/products/search?q=arroz&postalCode=28001'
```

La búsqueda usa datos de demostración durante la Fase 0. Su finalidad es fijar el contrato HTTP sin acoplar la API a una fuente real prematuramente.

## Base de datos

La migración `migrations/001_init.sql` introduce:

- `supermarkets`
- `supermarket_products`
- `price_observations`

Las observaciones son históricas e inmutables. Un cambio de precio crea una nueva fila.

## Convenciones para proveedores

Cada nuevo proveedor debe:

1. Implementar `internal/supermarkets.Provider`.
2. Mantener toda la lógica específica de la cadena aislada.
3. Normalizar unidades y moneda antes de devolver productos.
4. Conservar `external_id` y URL de origen cuando estén disponibles.
5. Admitir contexto/cancelación.
6. Tener pruebas con fixtures locales; los tests normales no deben depender de Internet.
7. Documentar la fuente y cualquier dependencia de código postal, tienda, cookies o sesión.

## Política de extracción

Antes de implementar un extractor real se investigará la fuente y se documentará la decisión. Preferencia técnica:

1. API o feed oficialmente ofrecido/documentado.
2. Respuesta estructurada accesible y cuyo uso sea apropiado.
3. HTML público y estable.
4. Automatización de navegador únicamente cuando sea necesaria.

No se introducirá evasión de controles de acceso, captchas o medidas antibot.

## Estrategia de ramas

Cada fase o cambio relevante se desarrolla en rama y se integra mediante pull request. Las decisiones de arquitectura que afecten al contrato o al modelo de datos se documentan en `docs/decisions/`.
