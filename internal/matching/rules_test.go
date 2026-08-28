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
        if matches[0].Source != SourceRulesV3 {
            t.Fatalf("%q: source=%q, want %q", tc.name, matches[0].Source, SourceRulesV3)
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

func TestSuggestMatchesFreshDIAProduceBySourceCategory(t *testing.T) {
    cases := []struct {
        categoryID        string
        normalizedCategory string
        name              string
        want              string
    }{
        {"L2022", "food.produce.vegetable", "Cebolla dulce granel 1 Kg aprox.", "cebolla"},
        {"L2022", "food.produce.vegetable", "Cebolla morada malla 500 g", "cebolla_morada"},
        {"L2023", "food.produce.vegetable", "Tomate pera bandeja 500 g", "tomate"},
        {"L2023", "food.produce.vegetable", "Tomates cherry bandeja 400 g", "tomate_cherry"},
        {"L2023", "food.produce.vegetable", "Tomate cherry pera bandeja 250 g", "tomate_cherry"},
        {"L2023", "food.produce.vegetable", "Pimiento rojo unidad 350 g aprox.", "pimiento_rojo"},
        {"L2023", "food.produce.vegetable", "Pimiento verde freír granel 500 g aprox.", "pimiento"},
        {"L2023", "food.produce.vegetable", "Pepino granel 1 Kg aprox.", "pepino"},
        {"L2027", "food.produce.vegetable", "Brotes tiernos bandeja 100 g", "brotes_tiernos"},
        {"L2027", "food.produce.vegetable", "Canónigos Dia Vegecampo 70 g", "canonigos"},
        {"L2028", "food.produce.vegetable", "Patatas para freír 2 Kg", "patata"},
        {"L2181", "food.produce.vegetable", "Calabaza cortada a trozos al vacío 500 g", "calabaza"},
        {"L2029", "food.produce.mushroom", "Champiñón laminado bandeja 250 g", "champinon"},
        {"L2029", "food.produce.mushroom", "Seta ostra bandeja 200 g", "seta"},
    }

    for _, tc := range cases {
        product := catalog.Product{
            SupermarketID:        "dia",
            ExternalID:           "sku",
            Name:                 tc.name,
            SourceCategoryID:     tc.categoryID,
            ItemType:             "food_ingredient",
            NormalizedCategory:   tc.normalizedCategory,
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
        }
        matches := Suggest(product)
        if len(matches) != 1 || matches[0].CanonicalIngredientID != tc.want {
            t.Fatalf("%s %q: got %#v, want %s", tc.categoryID, tc.name, matches, tc.want)
        }
        if matches[0].Score != 0.99 || matches[0].Status != "automatic" || matches[0].Source != SourceRulesV3 {
            t.Fatalf("%q: unexpected match metadata %#v", tc.name, matches[0])
        }
    }
}

func TestSuggestLeavesSpringOnionUnmatchedFromGenericCebolla(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "spring-onion",
        Name:                 "Cebolla tierna unidad",
        SourceCategoryID:     "L2022",
        ItemType:             "food_ingredient",
        NormalizedCategory:   "food.produce.vegetable",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("spring onion unexpectedly collapsed into generic cebolla: %#v", matches)
    }
}

func TestSuggestRejectsMicrowaveProduceEvenUnderFreshCategoryID(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "microwave-broccoli",
        Name:                 "Brócoli para microondas Florette 225 g",
        SourceCategoryID:     "L2024",
        ItemType:             "food_ingredient",
        NormalizedCategory:   "food.produce.vegetable",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("microwave produce unexpectedly matched: %#v", matches)
    }
}

func TestSuggestVegetableRequiresVerifiedFreshCategory(t *testing.T) {
    product := catalog.Product{
        SupermarketID:        "dia",
        ExternalID:           "frozen-onion",
        Name:                 "Cebolla troceada Dia Vegecampo 400 g",
        SourceCategoryID:     "L2025",
        ItemType:             "food_ingredient",
        NormalizedCategory:   "food.produce.vegetable",
        RecipeCompatible:     true,
        ClassificationStatus: "classified",
    }
    if matches := Suggest(product); len(matches) != 0 {
        t.Fatalf("non-fresh source category unexpectedly matched: %#v", matches)
    }
}

func TestSuggestProduceRequiresExpectedNormalizedCategory(t *testing.T) {
    cases := []catalog.Product{
        {
            SupermarketID:        "dia",
            ExternalID:           "wrong-veg-classification",
            Name:                 "Tomate pera bandeja 500 g",
            SourceCategoryID:     "L2023",
            ItemType:             "food_ingredient",
            NormalizedCategory:   "food.prepared",
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
        },
        {
            SupermarketID:        "dia",
            ExternalID:           "wrong-mushroom-classification",
            Name:                 "Champiñón entero bandeja 250 g",
            SourceCategoryID:     "L2029",
            ItemType:             "food_ingredient",
            NormalizedCategory:   "food.produce.vegetable",
            RecipeCompatible:     true,
            ClassificationStatus: "classified",
        },
    }
    for _, product := range cases {
        if matches := Suggest(product); len(matches) != 0 {
            t.Fatalf("%q wrong normalized category unexpectedly matched: %#v", product.Name, matches)
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
