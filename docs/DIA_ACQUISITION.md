# Estrategia de adquisición — DIA

## Estado

Documento de trabajo de la Fase 1. La adquisición debe mantenerse separada del contrato REST que consumirán las aplicaciones.

## Decisión

No se implementará la búsqueda pública de nuestra API haciendo peticiones en tiempo real al buscador de DIA.

La estrategia objetivo es:

```text
sitemap / páginas de categoría públicas
            ↓
        prices-worker
            ↓
     RawProduct DIA
            ↓
       Normalize()
            ↓
       PostgreSQL
            ↓
GET /api/v1/products/search
```

La búsqueda que use `Gestor-Alimentacion` se resolverá contra nuestra base de datos.

## Motivo

A fecha 2026-08-17 el `robots.txt` público de DIA:

- publica `https://www.dia.es/sitemap.xml` para descubrimiento;
- bloquea expresamente rutas `/search?...` y `/search/reduced?...`;
- bloquea `/products/`;
- permite la indexación de numerosas páginas de categoría/listado con restricciones específicas de paginación y filtros.

Por tanto, el crawler no debe basarse en los endpoints/rutas de búsqueda bloqueados.

## Datos observables en listados públicos

Las páginas de categoría públicas muestran suficiente semántica para el modelo inicial:

- identificador SKU visible en el contenido renderizado (`sku_id`);
- nombre del producto;
- formato/cantidad en el nombre/listado;
- precio regular;
- precio por unidad (`KILO`, `LITRO`, `UNIDAD`, etc.);
- promoción Club DIA cuando existe;
- porcentaje de descuento cuando existe;
- precio promocional cuando existe;
- estado de disponibilidad mediante acciones como `Añadir` / `Agotado`.

Ejemplo observado en páginas públicas de arroz: SKU `250021`, `Vasos de arroz Dia Al Punto 2 x 125 g`, precio normal, descuento Club DIA y precio por kilo.

## Código postal

DIA indica públicamente que puede adaptar el surtido al código postal. La adquisición deberá tratar la ubicación como dimensión del snapshot cuando sea necesaria.

No se asumirá que un producto o precio observado sin ubicación explícita representa todo el territorio nacional.

## Provider vs crawler

`internal/supermarkets/dia.Provider` no conoce HTML ni URLs de DIA. Recibe `RawProduct` a través de una interfaz `Source` y los normaliza.

El crawler/worker futuro será responsable de:

1. descubrir URLs de catálogo permitidas;
2. descargar con límites de frecuencia y timeouts;
3. extraer campos DIA;
4. producir `RawProduct`;
5. persistir producto + observación de precio;
6. registrar métricas de encontrados/normalizados/descartados/errores.

Esto permite cambiar el mecanismo de adquisición sin cambiar la API pública.

## Qué no haremos

- No scraping desde el frontend.
- No consultar DIA por cada búsqueda de usuario.
- No usar rutas bloqueadas por `robots.txt` como base del crawler.
- No asumir que un endpoint interno de la web es una API pública.
- No autenticar cuentas reales de Club DIA en el worker.

## Próximo paso técnico

Implementar un `ListingSource` basado en snapshots/fixtures de páginas de categoría y tests de parsing. Después conectar el fetch HTTP únicamente a URLs de catálogo que hayamos validado y documentado.
