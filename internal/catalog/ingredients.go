package catalog

import "context"

type CanonicalIngredient struct {
    ID          string `json:"id"`
    Name        string `json:"name"`
    Category    string `json:"category,omitempty"`
    Subtype     string `json:"subtype,omitempty"`
    DefaultUnit string `json:"defaultUnit,omitempty"`
}

type IngredientProduct struct {
    Product     Product `json:"product"`
    MatchStatus string  `json:"matchStatus"`
    MatchScore  float64 `json:"matchScore,omitempty"`
    MatchSource string  `json:"matchSource,omitempty"`
}

type IngredientStore interface {
    Ingredients(ctx context.Context) ([]CanonicalIngredient, error)
    SearchIngredients(ctx context.Context, query string) ([]CanonicalIngredient, error)
    IngredientProducts(ctx context.Context, ingredientID, postalCode string) ([]IngredientProduct, error)
}
