package matching

import (
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestSuggestMatchesRawRiceVariants(t *testing.T) {
    cases := []struct {
        name string
        want string
    }{
        {"Arroz redondo SOS 1 Kg", "arroz_redondo"},
        {"Arroz extra Dia Arrozona 1 Kg", "arroz_extra"},
        {"Arroz vaporizado Dia Arrozona 1 Kg", "arroz_vaporizado"},
        {"Arroz basmati marca X 1 Kg", "arroz_basmati"},
        {"Arroz integral marca X 1 Kg", "arroz_integral"},
    }

    for _, tc := range cases {
        matches := Suggest(catalog.Product{SupermarketID: "dia", ExternalID: "sku", Name: tc.name})
        if len(matches) != 1 || matches[0].CanonicalIngredientID != tc.want {
            t.Fatalf("%q: got %#v, want %s", tc.name, matches, tc.want)
        }
        if matches[0].Source != SourceRulesV2 {
            t.Fatalf("%q: source=%q, want %q", tc.name, matches[0].Source, SourceRulesV2)
        }
    }
}

func TestSuggestRejectsPreparedRice(t *testing.T) {
    names := []string{
        "Arroz tres delicias Dia Al Punto 850 g",
        "Vasos de arroz integral Brillante 2 x 125 g",
        "Arroz de marisco Dia Al Punto 330 g",
        "Arroz de secreto ibérico y setas Selección de Dia 350 g",
        "Arroz con pollo 350 g",
    }
    for _, name := range names {
        if matches := Suggest(catalog.Product{SupermarketID: "dia", ExternalID: "sku", Name: name}); len(matches) != 0 {
            t.Fatalf("%q unexpectedly matched: %#v", name, matches)
        }
    }
}

func TestSuggestMatchesBasicMilkVariantsAndPreservesLactoseModifier(t *testing.T) {
    cases := []struct {
        name string
        want string
    }{
        {"Leche entera Dia Lactea 1 L", "leche_entera"},
        {"Leche semidesnatada Dia Lactea 1 L", "leche_semidesnatada"},
        {"Leche desnatada Dia Lactea 1 L", "leche_desnatada"},
        {"Leche entera sin lactosa marca X 1 L", "leche_entera_sin_lactosa"},
        {"Leche semidesnatada sin lactosa marca X 1 L", "leche_semidesnatada_sin_lactosa"},
        {"Leche desnatada sin lactosa marca X 1 L", "leche_desnatada_sin_lactosa"},
    }

    for _, tc := range cases {
        product := catalog.Product{
            SupermarketID:        "dia",
            ExternalID:           "sku",
            Name:                 tc.name,
            ItemType:             "food_ingredient",
            NormalizedCategory:   "food.dairy.milk",
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
        }
        matches := Suggest(product)
        if len(matches) != 1 || matches[0].CanonicalIngredientID != tc.want {
            t.Fatalf("%q: got %#v, want %s", tc.name, matches, tc.want)
        }
    }
}

func TestSuggestLeavesSpecialMilkUnmatched(t *testing.T) {
    names := []string{
        "Leche condensada 397 g",
        "Leche evaporada 500 ml",
        "Leche semidesnatada con calcio 1 L",
        "Leche fresca entera pasteurizada 1 L",
        "Leche infantil de crecimiento 1 L",
    }
    for _, name := range names {
        product := catalog.Product{
            SupermarketID:        "dia",
            ExternalID:           "sku",
            Name:                 name,
            ItemType:             "food_ingredient",
            NormalizedCategory:   "food.dairy.milk",
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
        }
        if matches := Suggest(product); len(matches) != 0 {
            t.Fatalf("%q unexpectedly matched: %#v", name, matches)
        }
    }
}

func TestSuggestRespectsExplicitNonRecipeClassification(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "prepared-rice",
        Name:                 "Arroz redondo especial 300 g",
        ItemType:             "prepared_food",
        RecipeCompatible:     false,
        ClassificationStatus: "classified",
    }

    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("classified prepared product unexpectedly matched: %#v", matches)
    }
}

func TestSuggestRejectsClassifiedProductFromWrongCategory(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "wrong-category",
        Name:                 "Leche entera 1 L",
        ItemType:             "food_ingredient",
        NormalizedCategory:   "food.prepared",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("wrong-category product unexpectedly matched: %#v", matches)
    }
}
