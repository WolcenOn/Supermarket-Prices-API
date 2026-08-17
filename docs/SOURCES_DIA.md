# Fuente DIA — notas de integración

## Objetivo

Documentar la estrategia de adquisición para DIA antes de acoplar el servicio a una fuente concreta.

## Hallazgos públicos

La web pública de DIA expone en sus páginas de catálogo y ofertas información suficiente para nuestro modelo inicial:

- nombre de producto;
- formato/cantidad;
- precio normal;
- precio promocional cuando aplica;
- precio por unidad (`€/kg`, `€/l`, `€/unidad`, etc.);
- promociones Club DIA y promociones exclusivas online;
- disponibilidad visual (`Añadir` / `Agotado`).

DIA también indica que puede adaptar el surtido según código postal, por lo que `postalCode` debe formar parte del contexto de adquisición y de la observación de precio.

## Decisión de implementación

No se considerará automáticamente pública una API o endpoint JSON utilizado internamente por `dia.es`. La integración seguirá este orden de preferencia:

1. API/feed oficialmente documentado, si existe y es aplicable.
2. Datos estructurados públicos presentes en las páginas del catálogo.
3. Parsing de HTML público con límites de frecuencia y cache.
4. Automatización de navegador solo si las opciones anteriores no permiten obtener el catálogo necesario.

El mecanismo concreto se encapsula dentro de `internal/supermarkets/dia`; el resto del servicio solo conoce la interfaz `supermarkets.Provider`.

## Promociones

Se distinguirán al menos:

- `regular`: precio ordinario;
- `club`: precio/promoción Club DIA;
- `online`: promoción exclusiva online;
- `multibuy`: promociones como 2ª unidad al 50 %, 3x2, N unidades por importe, etc.

Una promoción compleja no debe convertirse de forma ingenua en un precio unitario si depende de comprar múltiples unidades. Se conservará el texto de la promoción y, en una fase posterior, el optimizador será responsable de evaluarla según la cantidad requerida.

## Ubicación

Las observaciones deben poder asociarse a `postal_code`. No asumiremos que un precio o surtido observado sin ubicación sea válido para toda España.

## Pruebas

Los tests del parser/normalizador usarán fixtures locales derivados de estructuras observadas públicamente. Los tests automáticos no dependerán de la disponibilidad de `dia.es`.

## Riesgos

- cambios frecuentes en HTML/estructura de la web;
- promociones que no son reducibles a un único precio;
- productos de peso aproximado;
- diferencias de catálogo y precio por ubicación;
- límites técnicos o condiciones de uso de la fuente.

Cualquier cambio de mecanismo de adquisición debe actualizar este documento y mantener estable el modelo normalizado expuesto al resto del servicio.
