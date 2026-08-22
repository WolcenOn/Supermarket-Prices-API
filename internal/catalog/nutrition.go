package catalog

import "time"

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
