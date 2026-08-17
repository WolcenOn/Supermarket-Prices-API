# ADR 0001 — Servicio de precios independiente

## Estado

Aceptado.

## Contexto

`Gestor-Alimentacion` necesita comparar el coste de su lista de compra entre supermercados y, en fases posteriores, respetar preferencias de marca/calidad y limitar el número de tiendas. La adquisición de precios cambia con frecuencia y tiene un ciclo de vida distinto al de usuarios, hogares, recetas y planes.

## Decisión

El sistema de precios será un producto independiente con repositorio, API y PostgreSQL propios. `Gestor-Alimentacion` lo consumirá mediante REST.

El servicio de precios será agnóstico al origen de la lista: no conocerá recetas ni hogares. Sus entradas serán productos/necesidades normalizadas y restricciones de comparación.

## Consecuencias

### Positivas
- Los cambios en extractores no fuerzan despliegues del gestor.
- El catálogo puede reutilizarse desde otros clientes.
- El histórico de precios puede crecer sin contaminar la base de datos del gestor.
- API y worker se pueden escalar por separado.

### Costes
- Hay dos servicios y dos bases de datos que operar.
- Debemos definir contratos REST y versionarlos.
- La integración requiere resolver matching entre ingredientes y productos comerciales.
