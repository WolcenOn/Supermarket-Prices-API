package classification

import (
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const SourceRulesV1 = "rules:v1"

type Result struct {
    ItemType           string
    NormalizedCategory string
    RecipeCompatible   bool
    Status             string
    Score              float64
    Source             string
}

var accentReplacer = strings.NewReplacer(
    "á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
)

var preparedTerms = []string{
    "plato preparado",
    "platos preparados",
    "comida preparada",
    "listo para comer",
    "listo para consumir",
    "tres delicias",
    "secreto iberico",
    "risotto",
    "paella preparada",
    "vaso de arroz",
    "vasos de arroz",
    "vasito de arroz",
    "vasitos de arroz",
}

var beverageTerms = []string{
    "bebidas",
    "refresco",
    "refrescos",
    "agua mineral",
    "cerveza",
    "vino",
    "zumo",
    "zumos",
    "batido",
    "batidos",
}

var householdTerms = []string{
    "limpieza",
    "hogar",
    "detergente",
    "lavavajillas",
    "papel higienico",
    "papel de cocina",
    "suavizante",
}

var knownRiceTerms = []string{
    "arroz redondo",
    "arroz extra",
    "arroz vaporizado",
    "arroz basmati",
    "arroz integral",
    "arroz largo",
}

// Classify assigns a conservative product type before canonical matching.
// Source taxonomy is preferred when available, while product-name rules are
// used as a fallback. Unknown products stay pending instead of being forced
// into a recipe-compatible category.
func Classify(product catalog.Product) Result {
    name := normalize(product.Name)
    sourceCategory := normalize(strings.Join([]string{
        product.SourceCategoryID,
        product.SourceCategoryName,
        product.SourceCategoryPath,
    }, " "))
    combined := strings.TrimSpace(sourceCategory + " " + name)

    if containsAny(combined, preparedTerms) {
        return classified("prepared_food", "food.prepared", false, 0.99)
    }

    if containsAny(combined, householdTerms) {
        return classified("household", "household", false, 0.98)
    }

    if containsAny(sourceCategory, beverageTerms) || containsAny(name, beverageTerms) {
        return classified("beverage", "beverage", false, 0.96)
    }

    if strings.Contains(name, "arroz") && (strings.Contains(sourceCategory, "arroz") || containsAny(name, knownRiceTerms)) {
        return classified("food_ingredient", "food.pantry.cereal.rice", true, 0.98)
    }

    if containsAny(name, knownRiceTerms) {
        return classified("food_ingredient", "food.pantry.cereal.rice", true, 0.94)
    }

    return Result{
        ItemType:           "other",
        NormalizedCategory: "",
        RecipeCompatible:   false,
        Status:             "pending",
        Score:              0,
        Source:             SourceRulesV1,
    }
}

func Apply(product *catalog.Product) Result {
    if product == nil {
        return Result{}
    }
    result := Classify(*product)
    product.ItemType = result.ItemType
    product.NormalizedCategory = result.NormalizedCategory
    product.RecipeCompatible = result.RecipeCompatible
    product.ClassificationStatus = result.Status
    product.ClassificationScore = result.Score
    product.ClassificationSource = result.Source
    return result
}

func classified(itemType, category string, recipeCompatible bool, score float64) Result {
    return Result{
        ItemType:           itemType,
        NormalizedCategory: category,
        RecipeCompatible:   recipeCompatible,
        Status:             "classified",
        Score:              score,
        Source:             SourceRulesV1,
    }
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
    value = strings.ToLower(strings.TrimSpace(value))
    value = accentReplacer.Replace(value)
    value = strings.NewReplacer("-", " ", "_", " ", "/", " ").Replace(value)
    return strings.Join(strings.Fields(value), " ")
}
