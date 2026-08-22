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
            if got.Source != SourceRulesV1 {
                t.Fatalf("Source = %q, want %q", got.Source, SourceRulesV1)
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
}
