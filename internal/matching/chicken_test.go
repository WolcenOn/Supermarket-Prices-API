package matching

import (
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestSuggestMatchesFreshDIAChickenBreast(t *testing.T) {
    cases := []string{
        "Pechugas enteras de pollo Selección de Dia 700 g aprox.",
        "Pechugas fileteadas Selección de Dia bandeja 650 g aprox.",
        "Pechuga de pollo entera formato familiar Selección de Dia 1.3 Kg aprox.",
    }
    for _, name := range cases {
        product := catalog.Product{
            SupermarketID:        "dia",
            ExternalID:           "sku",
            Name:                 name,
            SourceCategoryID:     "L2202",
            ItemType:             "food_ingredient",
            NormalizedCategory:   "food.meat.chicken_breast",
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
        }
        matches := Suggest(product)
        if len(matches) != 1 || matches[0].CanonicalIngredientID != "pechuga_pollo" {
            t.Fatalf("%q: got %#v", name, matches)
        }
        if matches[0].Score != 0.99 || matches[0].Status != "automatic" || matches[0].Source != SourceRulesV3 {
            t.Fatalf("%q: unexpected metadata %#v", name, matches[0])
        }
    }
}

func TestSuggestDoesNotMatchChickenBreastOutsideFreshCategory(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "charcuterie",
        Name:                 "Pechuga de pollo 95% Dia Nuestra Alacena 120 g",
        SourceCategoryID:     "L2342",
        ItemType:             "food_ingredient",
        NormalizedCategory:   "food.meat.chicken_breast",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("charcuterie chicken unexpectedly matched: %#v", matches)
    }
}

func TestSuggestRejectsPreparedChickenBreast(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "prepared",
        Name:                 "Pechuga de pollo empanada Selección de Dia 600 g",
        SourceCategoryID:     "L2202",
        ItemType:             "food_ingredient",
        NormalizedCategory:   "food.meat.chicken_breast",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("prepared chicken breast unexpectedly matched: %#v", matches)
    }
}
