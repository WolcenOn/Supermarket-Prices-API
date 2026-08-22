package dia

import (
    "testing"
)

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
    if details.IngredientsText != "Arroz redondo." {
        t.Fatalf("unexpected ingredients %q", details.IngredientsText)
    }
    if details.Nutrition == nil {
        t.Fatal("expected nutrition")
    }
    nutrition := details.Nutrition
    if nutrition.BasisAmount != 100 || nutrition.BasisUnit != "g" || nutrition.EnergyKcal != 345 || nutrition.CarbohydratesG != 77 || nutrition.ProteinG != 7 {
        t.Fatalf("unexpected nutrition: %+v", *nutrition)
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
