package matching

import (
    "strings"

    canonicaltext "github.com/WolcenOn/Supermarket-Prices-API/internal/canonical"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const SourceRulesV2 = "rules:v2"

type Match struct {
    CanonicalIngredientID string
    SupermarketID         string
    ExternalID            string
    Score                 float64
    Status                string
    Source                string
}

type rule struct {
    ingredientID string
    phrase       string
    score        float64
}

var riceRules = []rule{
    {ingredientID: "arroz_redondo", phrase: "arroz redondo", score: 0.99},
    {ingredientID: "arroz_extra", phrase: "arroz extra", score: 0.99},
    {ingredientID: "arroz_vaporizado", phrase: "arroz vaporizado", score: 0.99},
    {ingredientID: "arroz_basmati", phrase: "arroz basmati", score: 0.99},
    {ingredientID: "arroz_integral", phrase: "arroz integral", score: 0.99},
}

// Vegetable rules are scoped to verified fresh DIA source categories. The
// category gate is as important as the phrase: the same words may appear in
// frozen, canned or prepared products, which must not become automatic recipe
// ingredient matches.
var vegetableRulesBySourceCategory = map[string][]rule{
    "l2022": {
        {ingredientID: "ajo", phrase: "ajo", score: 0.99},
        {ingredientID: "cebolla", phrase: "cebolla", score: 0.99},
        {ingredientID: "puerro", phrase: "puerro", score: 0.99},
    },
    "l2023": {
        {ingredientID: "tomate", phrase: "tomate", score: 0.99},
        {ingredientID: "pimiento", phrase: "pimiento", score: 0.99},
        {ingredientID: "pepino", phrase: "pepino", score: 0.99},
    },
    "l2024": {
        {ingredientID: "brocoli", phrase: "brocoli", score: 0.99},
        {ingredientID: "coliflor", phrase: "coliflor", score: 0.99},
        {ingredientID: "judias_verdes", phrase: "judias verdes", score: 0.99},
        {ingredientID: "judias_verdes", phrase: "judia verde", score: 0.99},
    },
    "l2027": {
        {ingredientID: "brotes_tiernos", phrase: "brotes tiernos", score: 0.99},
        {ingredientID: "lechuga", phrase: "lechuga", score: 0.99},
        {ingredientID: "espinaca", phrase: "espinaca", score: 0.99},
    },
    "l2028": {
        {ingredientID: "patata", phrase: "patata", score: 0.99},
        {ingredientID: "zanahoria", phrase: "zanahoria", score: 0.99},
    },
    "l2181": {
        {ingredientID: "calabacin", phrase: "calabacin", score: 0.99},
        {ingredientID: "calabaza", phrase: "calabaza", score: 0.99},
        {ingredientID: "berenjena", phrase: "berenjena", score: 0.99},
    },
    "l2029": {
        {ingredientID: "champinon", phrase: "champinon", score: 0.99},
        {ingredientID: "seta", phrase: "seta", score: 0.99},
    },
}

var preparedRiceTerms = []string{
    "tres delicias",
    "al punto",
    "vasos de arroz",
    "vaso de arroz",
    "vasitos de arroz",
    "vasito de arroz",
    "arroz con ",
    "marisco",
    "secreto iberico",
    "risotto",
    "paella preparada",
}

// Milk exclusions are deliberately conservative. These products may still be
// useful groceries, but their extra semantic modifiers mean they should not be
// collapsed into a basic recipe-level milk canonical without a dedicated rule.
var specialMilkTerms = []string{
    "condensada",
    "evaporada",
    "en polvo",
    "crecimiento",
    "continuacion",
    "infantil",
    "preparado lacteo",
    "batido",
    "chocolate",
    "cacao",
    "vainilla",
    "fresa",
    "cafe",
    "proteina",
    "calcio",
    "omega",
    "fresca",
    "pasteurizada",
}

// Suggest returns only high-confidence, recipe-level equivalences. Ambiguous
// products intentionally remain unmatched for later review instead of forcing a
// potentially wrong canonical ingredient.
func Suggest(product catalog.Product) []Match {
    // New imports are classified before matching. A classified non-recipe item
    // must never be promoted to a canonical recipe ingredient just because its
    // commercial name contains the same words. Legacy/unclassified products
    // keep the previous matching behavior until they are re-imported.
    if product.ClassificationStatus != "" && !product.RecipeCompatible {
        return nil
    }

    name := normalize(product.Name)
    if name == "" {
        return nil
    }

    if match, ok := suggestVegetable(product, name); ok {
        return []Match{match}
    }
    if match, ok := suggestMilk(product, name); ok {
        return []Match{match}
    }
    if match, ok := suggestRice(product, name); ok {
        return []Match{match}
    }
    return nil
}

func suggestVegetable(product catalog.Product, name string) (Match, bool) {
    // Unlike the older rice fallback, vegetable matching is intentionally
    // classification-dependent. This prevents names from unrelated legacy
    // products from becoming automatic produce matches.
    if product.ClassificationStatus != "classified" ||
        !product.RecipeCompatible ||
        product.ItemType != "food_ingredient" ||
        product.NormalizedCategory != "food.produce.vegetable" {
        return Match{}, false
    }

    categoryID := strings.ToLower(strings.TrimSpace(product.SourceCategoryID))
    candidates, ok := vegetableRulesBySourceCategory[categoryID]
    if !ok {
        return Match{}, false
    }

    // DIA's "cebolla tierna" is a distinct culinary concept (spring onion),
    // so keep it unmatched until it has its own canonical instead of collapsing
    // it into the generic bulb-onion concept.
    if categoryID == "l2022" && strings.Contains(name, "cebolla tierna") {
        return Match{}, false
    }

    for _, candidate := range candidates {
        if strings.Contains(name, candidate.phrase) {
            return newMatch(product, candidate.ingredientID, candidate.score), true
        }
    }
    return Match{}, false
}

func suggestRice(product catalog.Product, name string) (Match, bool) {
    if isPreparedRice(name) {
        return Match{}, false
    }
    if product.ClassificationStatus == "classified" &&
        product.NormalizedCategory != "" &&
        product.NormalizedCategory != "food.pantry.cereal.rice" {
        return Match{}, false
    }

    for _, candidate := range riceRules {
        if strings.Contains(name, candidate.phrase) {
            return newMatch(product, candidate.ingredientID, candidate.score), true
        }
    }
    return Match{}, false
}

func suggestMilk(product catalog.Product, name string) (Match, bool) {
    if !strings.Contains(name, "leche") {
        return Match{}, false
    }
    if product.ClassificationStatus == "classified" &&
        product.NormalizedCategory != "food.dairy.milk" {
        return Match{}, false
    }
    if containsAny(name, specialMilkTerms) {
        return Match{}, false
    }

    var canonicalID string
    switch {
    case strings.Contains(name, "semidesnatada") || strings.Contains(name, "semi desnatada"):
        canonicalID = "leche_semidesnatada"
    case strings.Contains(name, "desnatada"):
        canonicalID = "leche_desnatada"
    case strings.Contains(name, "entera"):
        canonicalID = "leche_entera"
    default:
        return Match{}, false
    }

    if strings.Contains(name, "sin lactosa") {
        canonicalID += "_sin_lactosa"
    }
    return newMatch(product, canonicalID, 0.99), true
}

func newMatch(product catalog.Product, canonicalID string, score float64) Match {
    return Match{
        CanonicalIngredientID: canonicalID,
        SupermarketID:         product.SupermarketID,
        ExternalID:            product.ExternalID,
        Score:                 score,
        Status:                "automatic",
        Source:                SourceRulesV2,
    }
}

func isPreparedRice(name string) bool {
    return containsAny(name, preparedRiceTerms)
}

func containsAny(value string, terms []string) bool {
    for _, term := range terms {
        if strings.Contains(value, term) {
            return true
        }
    }
    return false
}

func normalize(value string) string {
    return canonicaltext.NormalizeText(value)
}
