package classification

import (
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestClassifyFreshDIAChickenBreast(t *testing.T) {
    cases := []string{
        "Pechugas enteras de pollo Selección de Dia 700 g aprox.",
        "Pechugas fileteadas Selección de Dia bandeja 650 g aprox.",
        "Pechuga de pollo entera formato familiar Selección de Dia 1.3 Kg aprox.",
    }
    for _, name := range cases {
        got := Classify(catalog.Product{
            Name:               name,
            SourceCategoryID:   "L2202",
            SourceCategoryName: "Pollo",
            SourceCategoryPath: "carnes/pollo/c/L2202",
        })
        if got.ItemType != "food_ingredient" || got.NormalizedCategory != "food.meat.chicken_breast" || !got.RecipeCompatible || got.Status != "classified" {
            t.Fatalf("%q: unexpected classification %#v", name, got)
        }
    }
}

func TestClassifyChickenBreastRequiresFreshChickenCategory(t *testing.T) {
    got := Classify(catalog.Product{
        Name:               "Pechuga de pollo 95% Dia Nuestra Alacena 120 g",
        SourceCategoryID:   "L2342",
        SourceCategoryName: "Pavo y pollo",
        SourceCategoryPath: "charcuteria/pavo-y-pollo/c/L2342",
    })
    if got.Status != "pending" || got.RecipeCompatible || got.ItemType != "other" {
        t.Fatalf("charcuterie chicken breast must remain pending: %#v", got)
    }
}

func TestClassifyDoesNotCollapseOtherChickenCutsOrPreparedBreast(t *testing.T) {
    names := []string{
        "Alas de pollo partidas Selección de Dia 700 g aprox.",
        "Jamoncitos de pollo Selección de Dia 1 Kg aprox.",
        "Filetes de contramuslo de pollo Selección de Dia 600 g aprox.",
        "Pechuga de pollo empanada Selección de Dia 600 g",
    }
    for _, name := range names {
        got := Classify(catalog.Product{
            Name:               name,
            SourceCategoryID:   "L2202",
            SourceCategoryName: "Pollo",
            SourceCategoryPath: "carnes/pollo/c/L2202",
        })
        if got.Status != "pending" || got.RecipeCompatible {
            t.Fatalf("%q must remain pending: %#v", name, got)
        }
    }
}
