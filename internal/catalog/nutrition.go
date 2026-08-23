package catalog

import (
    "context"
    "encoding/json"
    "strings"
    "time"
)

// ProductNutrition is a sourced nutrition snapshot for one exact commercial
// product. Pointer nutrient values deliberately distinguish a published zero
// from a value that the source did not provide.
type ProductNutrition struct {
    ProductID              string    `json:"productId,omitempty"`
    SupermarketID          string    `json:"supermarketId,omitempty"`
    ExternalID             string    `json:"externalId,omitempty"`
    EAN                    string    `json:"ean,omitempty"`
    Source                 string    `json:"source"`
    SourceURL              string    `json:"sourceUrl,omitempty"`
    ObservedAt             time.Time `json:"observedAt"`
    DescriptionText        string    `json:"descriptionText,omitempty"`
    SourceIngredientsBlock string    `json:"sourceIngredientsBlock,omitempty"`
    IngredientsText        string    `json:"ingredientsText,omitempty"`
    ResponsibleText        string    `json:"responsibleText,omitempty"`
    BasisAmount            *float64  `json:"basisAmount,omitempty"`
    BasisUnit              string    `json:"basisUnit,omitempty"`
    EnergyKJ               *float64  `json:"energyKJ,omitempty"`
    EnergyKcal             *float64  `json:"energyKcal,omitempty"`
    FatG                   *float64  `json:"fatG,omitempty"`
    SaturatedFatG          *float64  `json:"saturatedFatG,omitempty"`
    CarbohydratesG         *float64  `json:"carbohydratesG,omitempty"`
    SugarsG                *float64  `json:"sugarsG,omitempty"`
    FiberG                 *float64  `json:"fiberG,omitempty"`
    ProteinG               *float64  `json:"proteinG,omitempty"`
    SaltG                  *float64  `json:"saltG,omitempty"`
}

// ProportionalCalculationReady reports whether the source declared a positive
// mass/volume basis that can safely be scaled to another compatible quantity.
// A nutrition snapshot can still be useful for display when this is false; the
// caller must not assume a conventional 100 g or 100 ml basis.
func (n ProductNutrition) ProportionalCalculationReady() bool {
    if n.BasisAmount == nil || *n.BasisAmount <= 0 {
        return false
    }

    switch strings.ToLower(strings.TrimSpace(n.BasisUnit)) {
    case "g", "kg", "ml", "l":
        return true
    default:
        return false
    }
}

// MarshalJSON adds a derived readiness flag without persisting it. This keeps
// source facts untouched while making the public read contract explicit about
// whether proportional nutrition calculations are safe.
func (n ProductNutrition) MarshalJSON() ([]byte, error) {
    type productNutritionAlias ProductNutrition
    return json.Marshal(struct {
        productNutritionAlias
        ProportionalCalculationReady bool `json:"proportionalCalculationReady"`
    }{
        productNutritionAlias:        productNutritionAlias(n),
        ProportionalCalculationReady: n.ProportionalCalculationReady(),
    })
}

// NutritionStore exposes sourced nutrition snapshots already persisted for an
// exact commercial product. It is read-only and deliberately returns every
// available source so callers can distinguish DIA data from future sources
// such as Open Food Facts.
type NutritionStore interface {
    ProductNutrition(ctx context.Context, productID string) ([]ProductNutrition, error)
}
