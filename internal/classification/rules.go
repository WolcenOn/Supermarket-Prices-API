package classification

import (
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const (
    SourceRulesV2 = "rules:v2"
    SourceRulesV3 = "rules:v3"
)

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
    "arroz con ",
    "arroz de marisco",
    "dia al punto",
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

var freshVegetableCategoryIDs = map[string]struct{}{
    "l2022": {}, // Ajos cebollas y puerros
    "l2023": {}, // Tomates pimientos y pepinos
    "l2024": {}, // Brocoli coliflor y judias verdes
    "l2027": {}, // Lechugas y hojas verdes
    "l2028": {}, // Patatas y zanahorias
    "l2181": {}, // Calabacin calabaza y berenjena
}

var nonFoodSourceRoots = []string{
    "higiene-y-cuidado-del-cuerpo",
    "cabello-y-perfumeria",
    "mascotas",
    "salud-y-parafarmacia",
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

    if isNonFoodSourceCategory(product) {
        return classified("non_food", "non_food", false, 0.99)
    }

    if containsAny(combined, householdTerms) {
        return classified("household", "household", false, 0.98)
    }

    if containsAny(sourceCategory, beverageTerms) || containsAny(name, beverageTerms) {
        return classified("beverage", "beverage", false, 0.96)
    }

    if isMilkSourceCategory(product) && strings.Contains(name, "leche") {
        return classified("food_ingredient", "food.dairy.milk", true, 0.99)
    }

    if category, ok := freshProduceCategory(product); ok {
        return classified("food_ingredient", category, true, 0.99)
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
        Source:             SourceRulesV3,
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
        Source:             SourceRulesV3,
    }
}

func isNonFoodSourceCategory(product catalog.Product) bool {
    path := strings.ToLower(strings.Trim(strings.TrimSpace(product.SourceCategoryPath), "/"))
    for _, root := range nonFoodSourceRoots {
        if path == root || strings.HasPrefix(path, root+"/") {
            return true
        }
    }
    return false
}

func isMilkSourceCategory(product catalog.Product) bool {
    categoryID := normalize(product.SourceCategoryID)
    categoryName := normalize(product.SourceCategoryName)
    categoryPath := normalize(product.SourceCategoryPath)

    return categoryID == "l2051" ||
        categoryName == "leche" ||
        strings.Contains(categoryPath, "leche c l2051")
}

func freshProduceCategory(product catalog.Product) (string, bool) {
    categoryID := normalize(product.SourceCategoryID)
    if _, ok := freshVegetableCategoryIDs[categoryID]; ok {
        return "food.produce.vegetable", true
    }

    switch categoryID {
    case "l2029": // Setas y champinones
        return "food.produce.mushroom", true
    case "l2031": // Hierbas aromaticas
        return "food.produce.herb", true
    default:
        return "", false
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
