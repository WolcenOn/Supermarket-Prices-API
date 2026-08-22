package dia

import "testing"

func TestParseProductDetailHTMLExtractsNutritionAndValidGTIN(t *testing.T) {
    document := `<!doctype html><html><head><script type="application/ld+json">{"gtin13":"8410830001016"}</script></head><body>
<h1>Arroz redondo SOS 1 Kg</h1>
<div>Valor nutricional</div><div>Valores por 100g</div>
<div>Valor energético</div><span>1464kJ</span><span>345kcal</span>
<div>Grasas</div><span>0,7g</span>
<div>de las cuales saturadas</div><span>0,2g</span>
<div>Hidratos de Carbono</div><span>77g</span>
<div>de los cuales azúcares</div><span>0,5g</span>
<div>Fibra</div><span>1,4g</span>
<div>Proteínas</div><span>7g</span>
<div>Sal</div><span>0,01g</span>
<h2>Ingredientes</h2><div>Arroz redondo.</div>
<h2>Conservación y utilización</h2><div>Conservar en lugar fresco y seco.</div>
<h2>Información del responsable</h2><div>Empresa ejemplo S.A.</div><div>Madrid, España.</div>
<div>1,88 €</div>
</body></html>`

    details := ParseProductDetailHTML(document)
    if details.Name != "Arroz redondo SOS 1 Kg" {
        t.Fatalf("unexpected name %q", details.Name)
    }
    if details.EAN != "8410830001016" {
        t.Fatalf("expected validated EAN, got %q", details.EAN)
    }
    if details.SourceIngredientsBlock != "Arroz redondo." || details.IngredientsText != "Arroz redondo." {
        t.Fatalf("unexpected ingredients semantics: %+v", details)
    }
    if details.Nutrition == nil || details.NutritionSource != "dia_product_page" {
        t.Fatalf("expected sourced nutrition, got %+v", details)
    }
    nutrition := details.Nutrition
    if value(nutrition.BasisAmount) != 100 || nutrition.BasisUnit != "g" || value(nutrition.EnergyKcal) != 345 || value(nutrition.CarbohydratesG) != 77 || value(nutrition.ProteinG) != 7 {
        t.Fatalf("unexpected nutrition: %+v", *nutrition)
    }
}

func TestParseProductDetailHTMLPreservesPublishedZeroVersusMissing(t *testing.T) {
    document := `<html><body>
<h1>Agua mineral</h1>
<div>Valor nutricional</div><div>Valores por 100ml</div>
<div>Valor energético</div><span>0kJ</span><span>0kcal</span>
<div>Grasas</div><span>0g</span>
<div>Sal</div><span>0,01g</span>
</body></html>`

    details := ParseProductDetailHTML(document)
    if details.Nutrition == nil {
        t.Fatal("expected nutrition")
    }
    if details.Nutrition.EnergyKcal == nil || value(details.Nutrition.EnergyKcal) != 0 {
        t.Fatalf("published zero kcal must remain present: %+v", details.Nutrition)
    }
    if details.Nutrition.FatG == nil || value(details.Nutrition.FatG) != 0 {
        t.Fatalf("published zero fat must remain present: %+v", details.Nutrition)
    }
    if details.Nutrition.ProteinG != nil {
        t.Fatalf("missing protein must remain absent, got %+v", details.Nutrition)
    }
}

func TestParseProductDetailHTMLKeepsDIAAttributeBlockSeparateFromIngredients(t *testing.T) {
    document := `<html><body>
<h1>Arroz vaporizado Dia Arrozona 1 Kg</h1>
<div>Valor nutricional</div><div>Valores por 100g</div>
<div>Valor energético</div><span>1410kJ</span><span>332kcal</span>
<div>Proteínas</div><span>7g</span>
<h2>Arroz parboliled categoria I</h2>
<h2>Ingredientes</h2><div>Tipo de arroz: Vaporizado y precocido</div>
<h2>Conservación y utilización</h2><div>Conservar en lugar fresco y seco.</div>
</body></html>`

    details := ParseProductDetailHTML(document)
    if details.DescriptionText != "Arroz parboliled categoria I" {
        t.Fatalf("unexpected description %q", details.DescriptionText)
    }
    if details.SourceIngredientsBlock != "Tipo de arroz: Vaporizado y precocido" {
        t.Fatalf("unexpected source block %q", details.SourceIngredientsBlock)
    }
    if details.IngredientsText != "" {
        t.Fatalf("DIA attribute block must not be treated as ingredient declaration: %q", details.IngredientsText)
    }
}

func TestParseProductDetailHTMLKeepsPreparedProductTypeSeparateFromIngredients(t *testing.T) {
    document := `<html><body>
<h1>Arroz de marisco Dia Al Punto 330 g</h1>
<h2>ARROZ CON MARISCO, SALSA CON TOMATE Y CALDO DE PESCADO. PRODUCTO ULTRACONGELADO.</h2>
<h2>Ingredientes</h2><div>Tipo de producto: Arroz</div>
<h2>Conservación y utilización</h2><div>Conservar congelado.</div>
</body></html>`

    details := ParseProductDetailHTML(document)
    if details.DescriptionText == "" {
        t.Fatal("expected prepared-product description")
    }
    if details.SourceIngredientsBlock != "Tipo de producto: Arroz" || details.IngredientsText != "" {
        t.Fatalf("unexpected prepared-product ingredient semantics: %+v", details)
    }
}

func TestParseProductDetailHTMLRejectsInvalidGTINCandidate(t *testing.T) {
    details := ParseProductDetailHTML(`<html><body><h1>Producto</h1><script>{"gtin13":"8410830001017"}</script></body></html>`)
    if details.EAN != "" {
        t.Fatalf("expected invalid GTIN to be ignored, got %q", details.EAN)
    }
}

func TestAttachProductSourceURLsUsesMatchingSKUAndDIAHost(t *testing.T) {
    products := []RawProduct{{ExternalID: "28809"}, {ExternalID: "21415"}}
    document := `<a href="/arroz-pastas-y-legumbres/arroz/p/28809?foo=bar">Vaporizado</a>
<a href="https://www.dia.es/arroz-pastas-y-legumbres/arroz/p/21415">Redondo</a>
<a href="https://evil.example/p/28809">No usar</a>`

    attachProductSourceURLs(document, "https://www.dia.es/arroz-pastas-y-legumbres/c/L106", products)

    if products[0].SourceURL != "https://www.dia.es/arroz-pastas-y-legumbres/arroz/p/28809" {
        t.Fatalf("unexpected source URL %q", products[0].SourceURL)
    }
    if products[1].SourceURL != "https://www.dia.es/arroz-pastas-y-legumbres/arroz/p/21415" {
        t.Fatalf("unexpected source URL %q", products[1].SourceURL)
    }
}

func value(input *float64) float64 {
    if input == nil {
        return 0
    }
    return *input
}
