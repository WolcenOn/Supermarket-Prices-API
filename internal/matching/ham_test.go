package matching

import (
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func diaHamProduct(name, categoryID string) catalog.Product {
    return catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "sku",
        Name:                 name,
        SourceCategoryID:     categoryID,
        ItemType:             "food_ingredient",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
}

func TestSuggestMatchesDIACookedHamInDedicatedCategory(t *testing.T) {
    cases := []string{
        "Jamón cocido extra Nuestra Alacena 150 g",
        "Jamón york en lonchas 200 g",
        "Jamón cocido reducido en sal 180 g",
    }

    for _, name := range cases {
        matches := Suggest(diaHamProduct(name, "L2001"))
        if len(matches) != 1 {
            t.Fatalf("%q: expected one match, got %#v", name, matches)
        }
        match := matches[0]
        if match.CanonicalIngredientID != "jamon_cocido" {
            t.Fatalf("%q: unexpected canonical %q", name, match.CanonicalIngredientID)
        }
        if match.Score != 0.99 || match.Status != "automatic" || match.Source != SourceRulesV4 {
            t.Fatalf("%q: unexpected metadata %#v", name, match)
        }
    }
}

func TestSuggestDoesNotCollapseOtherHamConceptsIntoCookedHam(t *testing.T) {
    cases := []string{
        "Jamón serrano reserva 200 g",
        "Jamón ibérico de cebo 100 g",
        "Jamón de pavo 200 g",
        "Jamón 150 g",
    }

    for _, name := range cases {
        if matches := Suggest(diaHamProduct(name, "L2001")); len(matches) != 0 {
            t.Fatalf("%q unexpectedly matched: %#v", name, matches)
        }
    }
}

func TestSuggestDoesNotMatchCookedHamOutsideDedicatedCategory(t *testing.T) {
    cases := []catalog.Product{
        diaHamProduct("Croquetas de jamón cocido 500 g", "L2135"),
        diaHamProduct("Pizza de jamón york y queso 400 g", "L2101"),
        diaHamProduct("Pasta rellena de jamón cocido 250 g", "L9999"),
    }

    for _, product := range cases {
        if matches := Suggest(product); len(matches) != 0 {
            t.Fatalf("%q unexpectedly matched: %#v", product.Name, matches)
        }
    }
}

func TestSuggestRequiresRecipeCompatibleClassifiedIngredientForCookedHam(t *testing.T) {
    product := diaHamProduct("Jamón cocido extra 150 g", "L2001")
    product.RecipeCompatible = false
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("non-recipe product unexpectedly matched: %#v", matches)
    }
}
