package dia

import (
    "testing"
    "time"
)

func TestParseRenderedSnapshotClubPromotion(t *testing.T) {
    observedAt := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
    snapshot := `
condition :: true index::0 initialLoadedItems::10 item.isVisible:: item.type:: item.sku_id::250021 item.data::
Vasos de arroz Dia Al Punto 2 x 125 g
Sin gluten
Oferta CLUB Dia
1,10 €
9% dto.
1,00 €
(4,00 €/KILO)
[Button: Añadir]
condition :: true index::1 initialLoadedItems::10 item.isVisible:: item.type:: item.sku_id::267141 item.data::
Air Fryer
Arroz tres delicias Dia Al Punto 850 g
Sin gluten
2,42 €
(2,85 €/KILO)
[Button: Añadir]
`

    products := ParseRenderedSnapshot(snapshot, "28001", observedAt)
    if len(products) != 2 {
        t.Fatalf("expected 2 products, got %d", len(products))
    }

    first := products[0]
    if first.ExternalID != "250021" {
        t.Fatalf("expected sku 250021, got %q", first.ExternalID)
    }
    if first.Name != "Vasos de arroz Dia Al Punto 2 x 125 g" {
        t.Fatalf("unexpected name %q", first.Name)
    }
    if first.RegularPrice != 1.10 || first.PromotionalPrice != 1.00 {
        t.Fatalf("unexpected prices regular=%v promo=%v", first.RegularPrice, first.PromotionalPrice)
    }
    if first.PricePerUnit != 4.00 || first.PriceUnit != "KILO" {
        t.Fatalf("unexpected unit price %v/%s", first.PricePerUnit, first.PriceUnit)
    }
    if first.PromotionType != "club" || first.DiscountPct != 9 {
        t.Fatalf("unexpected promotion %+v", first)
    }
    if first.PostalCode != "28001" || !first.Available {
        t.Fatalf("unexpected location/availability %+v", first)
    }

    second := products[1]
    if second.ExternalID != "267141" || second.RegularPrice != 2.42 {
        t.Fatalf("unexpected second product %+v", second)
    }
}

func TestParseRenderedSnapshotMarksSoldOutAndMultibuy(t *testing.T) {
    snapshot := `
condition :: true index::0 initialLoadedItems::10 item.isVisible:: item.type:: item.sku_id::999999 item.data::
Helado ejemplo 4 x 90 g
Oferta Exclusiva online
2ª UD 50% DTO EN SELECCIÓN
4,99 €
(13,86 €/KILO)
[Button: Agotado]
`

    products := ParseRenderedSnapshot(snapshot, "", time.Time{})
    if len(products) != 1 {
        t.Fatalf("expected 1 product, got %d", len(products))
    }
    p := products[0]
    if p.Available {
        t.Fatal("expected sold out product")
    }
    if p.PromotionType != "multibuy" {
        t.Fatalf("expected multibuy, got %q", p.PromotionType)
    }
    if p.PromotionLabel == "" {
        t.Fatal("expected promotion label")
    }
}
