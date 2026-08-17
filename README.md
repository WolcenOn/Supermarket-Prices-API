# Supermarket Prices API

Servicio independiente para recopilar, normalizar, almacenar y comparar precios de supermercados en España.

El objetivo es ofrecer una API estable que pueda consumir `Gestor-Alimentacion` y otros clientes sin acoplarlos a la forma en que cada supermercado publica sus datos.

## Estado

**Fase 0 — Bootstrap y contratos.**

Incluye actualmente:

- API HTTP en Go.
- `GET /health`.
- `GET /api/v1/version`.
- `GET /api/v1/supermarkets`.
- `GET /api/v1/products/search?q=...&postalCode=...` con datos de demostración.
- Modelo de producto y observación de precio.
- Interfaz común de `Provider` para supermercados.
- Migración PostgreSQL inicial.
- Dockerfile y configuración Railway.
- CI con tests y `go vet`.

Los datos de demostración no pretenden representar precios reales; solo fijan el contrato de la API antes de integrar una fuente externa.

## Arquitectura

```text
Supermercados
     ↓
Providers / worker
     ↓
Normalización
     ↓
PostgreSQL
     ↓
Supermarket Prices API
     ↓
Gestor-Alimentacion / otros consumidores
```

La API pública no realizará scraping durante una petición. La adquisición de datos se ejecutará en un proceso separado y la API consultará datos persistidos.

Consulta [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) para el diseño y [`docs/DEVELOPMENT.md`](docs/DEVELOPMENT.md) para trabajar en el proyecto.

## Ejecutar localmente

```bash
go run ./cmd/api
```

Después:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/version
curl http://localhost:8080/api/v1/supermarkets
curl 'http://localhost:8080/api/v1/products/search?q=arroz&postalCode=28001'
```

## Tests

```bash
go test ./...
go vet ./...
```

## Roadmap

1. **Fase 0:** contratos, arquitectura y base ejecutable.
2. **Fase 1:** primer proveedor real y persistencia de precios.
3. **Fase 2:** catálogo canónico y matching entre supermercados.
4. **Fase 3:** comparador y optimizador de cesta.
5. **Fase 4:** marcas, calidad y preferencias de supermercado.
6. **Fase 5:** integración con Gestor-Alimentacion.

El roadmap detallado se mantiene en los issues del repositorio para que cada fase tenga alcance y criterio de aceptación claros.

## Licencia

GPL-3.0.
