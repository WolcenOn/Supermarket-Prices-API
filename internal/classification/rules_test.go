package classification

import (
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestClassifyUsesSourceTaxonomyBeforeCanonicalMatching(t *testing.T) {
    tests := []struct {
        name         string
        product      catalog.Product
        wantType     string
        wantCategory string
        wantRecipe   bool
        wantStatus   string
    }{
        {
            name: "basic rice ingredient",
            product: catalog.Product{
                Name:               "Arroz redondo SOS 1 Kg",
                SourceCategoryName: "Arroz, pastas y legumbres",
                SourceCategoryPath: "arroz-pastas-y-legumbres/c/L106",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.pantry.cereal.rice",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "prepared rice is not basic rice",
            product: catalog.Product{
                Name:               "Arroz con secreto ibérico 300 g",
                SourceCategoryName: "Platos preparados",
            },
            wantType:     "prepared_food",
            wantCategory: "food.prepared",
            wantRecipe:   false,
            wantStatus:   "classified",
        },
        {
            name: "prepared seafood rice stays prepared inside broad rice category",
            product: catalog.Product{
                Name:               "Arroz de marisco Dia Al Punto 330 g",
                SourceCategoryName: "Arroz, pastas y legumbres",
            },
            wantType:     "prepared_food",
            wantCategory: "food.prepared",
            wantRecipe:   false,
            wantStatus:   "classified",
        },
        {
            name: "ready rice cup stays prepared",
            product: catalog.Product{
                Name:               "Vasito de arroz redondo listo para comer 250 g",
                SourceCategoryName: "Arroz, pastas y legumbres",
            },
            wantType:     "prepared_food",
            wantCategory: "food.prepared",
            wantRecipe:   false,
            wantStatus:   "classified",
        },
        {
            name: "dia milk is a recipe ingredient",
            product: catalog.Product{
                Name:               "Leche semidesnatada Dia Láctea 1 L",
                SourceCategoryID:   "L2051",
                SourceCategoryName: "Leche",
                SourceCategoryPath: "huevos-leche-y-mantequilla/leche/c/L2051",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.dairy.milk",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "pasteurized dia milk stays a recipe ingredient",
            product: catalog.Product{
                Name:               "Leche entera pasteurizada Dia Láctea 1 L",
                SourceCategoryID:   "L2051",
                SourceCategoryName: "Leche",
                SourceCategoryPath: "huevos-leche-y-mantequilla/leche/c/L2051",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.dairy.milk",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "milk mention outside milk taxonomy is not enough",
            product: catalog.Product{
                Name:               "Chocolate con leche 100 g",
                SourceCategoryName: "Chocolate",
            },
            wantType:     "other",
            wantCategory: "",
            wantRecipe:   false,
            wantStatus:   "pending",
        },
        {
            name: "milk-category shake remains beverage",
            product: catalog.Product{
                Name:               "Batido de leche y cacao 1 L",
                SourceCategoryID:   "L2051",
                SourceCategoryName: "Leche",
                SourceCategoryPath: "huevos-leche-y-mantequilla/leche/c/L2051",
            },
            wantType:     "beverage",
            wantCategory: "beverage",
            wantRecipe:   false,
            wantStatus:   "classified",
        },
        {
            name: "fresh tomato category is a recipe ingredient",
            product: catalog.Product{
                Name:               "Tomates cherry Tomati&co 200 g",
                SourceCategoryID:   "L2023",
                SourceCategoryName: "Tomates pimientos y pepinos",
                SourceCategoryPath: "verduras/tomates-pimientos-y-pepinos/c/L2023",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.produce.vegetable",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "fresh squash category is a recipe ingredient",
            product: catalog.Product{
                Name:               "Calabaza 1.6 Kg aprox.",
                SourceCategoryID:   "L2181",
                SourceCategoryName: "Calabacin calabaza y berenjena",
                SourceCategoryPath: "verduras/calabacin-calabaza-y-berenjena/c/L2181",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.produce.vegetable",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "mushroom category is a recipe ingredient",
            product: catalog.Product{
                Name:               "Seta ostra bandeja 200 g",
                SourceCategoryID:   "L2029",
                SourceCategoryName: "Setas y champinones",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.produce.mushroom",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "aromatic herb category is a recipe ingredient",
            product: catalog.Product{
                Name:               "Perejil 20 g",
                SourceCategoryID:   "L2031",
                SourceCategoryName: "Hierbas aromaticas",
            },
            wantType:     "food_ingredient",
            wantCategory: "food.produce.herb",
            wantRecipe:   true,
            wantStatus:   "classified",
        },
        {
            name: "vegetable preserves remain pending",
            product: catalog.Product{
                Name:               "Maíz dulce Dia Vegecampo 3 x 70 g",
                SourceCategoryID:   "L2026",
                SourceCategoryName: "Conservas de verduras",
            },
            wantType:     "other",
            wantCategory: "",
            wantRecipe:   false,
            wantStatus:   "pending",
        },
        {
            name: "prepared salads remain pending",
            product: catalog.Product{
                Name:               "Ensalada preparada 200 g",
                SourceCategoryID:   "L2030",
                SourceCategoryName: "Ensaladas y verduras preparadas",
            },
            wantType:     "other",
            wantCategory: "",
            wantRecipe:   false,
            wantStatus:   "pending",
        },
        {
            name: "frozen vegetables remain pending",
            product: catalog.Product{
                Name:               "Verduras congeladas 500 g",
                SourceCategoryID:   "L2025",
                SourceCategoryName: "Verduras congeladas y al vapor",
            },
            wantType:     "other",
            wantCategory: "",
            wantRecipe:   false,
            wantStatus:   "pending",
        },
        {
            name: "beverage",
            product: catalog.Product{
                Name:               "Agua mineral 1,5 L",
                SourceCategoryName: "Bebidas",
            },
            wantType:     "beverage",
            wantCategory: "beverage",
            wantRecipe:   false,
            wantStatus:   "classified",
        },
        {
            name: "household item",
            product: catalog.Product{
                Name:               "Detergente lavadora 30 dosis",
                SourceCategoryName: "Limpieza y hogar",
            },
            wantType:     "household",
            wantCategory: "household",
            wantRecipe:   false,
            wantStatus:   "classified",
        },
        {
            name:         "unknown stays pending",
            product:      catalog.Product{Name: "Artículo desconocido"},
            wantType:     "other",
            wantCategory: "",
            wantRecipe:   false,
            wantStatus:   "pending",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := Classify(tt.product)
            if got.ItemType != tt.wantType {
                t.Fatalf("ItemType = %q, want %q", got.ItemType, tt.wantType)
            }
            if got.NormalizedCategory != tt.wantCategory {
                t.Fatalf("NormalizedCategory = %q, want %q", got.NormalizedCategory, tt.wantCategory)
            }
            if got.RecipeCompatible != tt.wantRecipe {
                t.Fatalf("RecipeCompatible = %v, want %v", got.RecipeCompatible, tt.wantRecipe)
            }
            if got.Status != tt.wantStatus {
                t.Fatalf("Status = %q, want %q", got.Status, tt.wantStatus)
            }
            if got.Source != SourceRulesV2 {
                t.Fatalf("Source = %q, want %q", got.Source, SourceRulesV2)
            }
        })
    }
}

func TestApplyWritesClassificationToProduct(t *testing.T) {
    product := catalog.Product{
        Name:               "Arroz basmati 1 Kg",
        SourceCategoryName: "Arroz, pastas y legumbres",
    }

    Apply(&product)

    if product.ItemType != "food_ingredient" || !product.RecipeCompatible {
        t.Fatalf("unexpected classification: %#v", product)
    }
    if product.ClassificationStatus != "classified" || product.ClassificationScore <= 0 {
        t.Fatalf("classification metadata not written: %#v", product)
    }
    if product.ClassificationSource != SourceRulesV2 {
        t.Fatalf("ClassificationSource = %q, want %q", product.ClassificationSource, SourceRulesV2)
    }
}
