# Arquitectura

## Objetivo

Supermarket Prices API es un servicio independiente para recopilar, normalizar, almacenar y consultar precios de supermercados. El servicio no conoce recetas, menús ni hogares: recibe necesidades de producto/ingrediente y devuelve información de catálogo y precio.

## Separación de responsabilidades

- **API**: contrato estable para consumidores.
- **Catálogo**: modelo normalizado común a todos los supermercados.
- **Providers**: adaptadores de adquisición específicos por supermercado.
- **Persistencia**: catálogo e histórico de observaciones en PostgreSQL.
- **Worker**: proceso futuro encargado de refrescar datos fuera del ciclo de una petición HTTP.
- **Optimizer**: fase futura para calcular cestas en uno o varios supermercados.

## Modelo de dominio inicial

### Supermarket
Identidad estable de una cadena.

### Supermarket product
Representa un producto tal como lo vende una cadena concreta. No es todavía un producto canónico compartido entre cadenas.

### Price observation
Una observación inmutable del precio de un producto, opcionalmente asociada a código postal y fecha. No se sobrescribe el histórico.

## Flujo objetivo

```text
Proveedor supermercado
        ↓
Normalización
        ↓
Persistencia PostgreSQL
        ↓
API REST
        ↓
Gestor-Alimentacion / otros consumidores
```

La extracción no debe ejecutarse en el camino crítico de `GET /products/search`. La API consulta datos ya almacenados; los refrescos se realizan por un worker o trabajos programados.

## Contrato de Provider

Cada supermercado implementará una interfaz común. La fuente concreta puede variar —API oficial, feed documentado, HTML o automatización de navegador— pero los consumidores nunca deben depender de los formatos propios de la cadena.

Antes de integrar una fuente se revisarán sus condiciones técnicas y de uso. Un endpoint empleado internamente por una web no se considerará automáticamente una API pública.

## Ubicación

Los precios y catálogos pueden depender de tienda o código postal. Por eso el modelo permite asociar la observación a `postal_code` desde el comienzo.

## Evolución prevista

1. Primer proveedor real y persistencia.
2. Producto canónico y matching entre cadenas.
3. Comparación de cesta.
4. Restricciones y preferencias.
5. Integración con Gestor-Alimentacion.

## Decisiones que evitamos prematuramente

- No introducimos IA para matching en la primera fase.
- No optimizamos cestas hasta contar con datos fiables.
- No hacemos scraping desde el frontend.
- No mezclamos usuarios ni hogares con el servicio de precios.
