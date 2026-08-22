package dia

import (
    "fmt"
    "strings"
    "time"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

// ProductNutrition converts a probed DIA detail page into the supermarket-
// agnostic sourced nutrition model used by persistence. It refuses to produce
// a record when DIA did not expose structured nutrition.
func (details ProductDetails) ProductNutrition(observedAt time.Time) (catalog.ProductNutrition, error) {
    if details.Nutrition == nil {
        return catalog.ProductNutrition{}, fmt.Errorf("dia nutrition: structured nutrition is not available")
    }
    source := strings.TrimSpace(details.NutritionSource)
    if source == "" {
        return catalog.ProductNutrition{}, fmt.Errorf("dia nutrition: nutrition source is required")
    }
    externalID := strings.TrimSpace(details.ExternalID)
    if externalID == "" {
        return catalog.ProductNutrition{}, fmt.Errorf("dia nutrition: external id is required")
    }
    if observedAt.IsZero() {
        observedAt = time.Now().UTC()
    } else {
        observedAt = observedAt.UTC()
    }

    facts := details.Nutrition
    return catalog.ProductNutrition{
        SupermarketID:          supermarketID,
        ExternalID:             externalID,
        EAN:                    strings.TrimSpace(details.EAN),
        Source:                 source,
        SourceURL:              strings.TrimSpace(details.SourceURL),
        ObservedAt:             observedAt,
        DescriptionText:        strings.TrimSpace(details.DescriptionText),
        SourceIngredientsBlock: strings.TrimSpace(details.SourceIngredientsBlock),
        IngredientsText:        strings.TrimSpace(details.IngredientsText),
        ResponsibleText:        strings.TrimSpace(details.ResponsibleText),
        BasisAmount:            facts.BasisAmount,
        BasisUnit:              strings.TrimSpace(strings.ToLower(facts.BasisUnit)),
        EnergyKJ:               facts.EnergyKJ,
        EnergyKcal:             facts.EnergyKcal,
        FatG:                   facts.FatG,
        SaturatedFatG:          facts.SaturatedFatG,
        CarbohydratesG:         facts.CarbohydratesG,
        SugarsG:                facts.SugarsG,
        FiberG:                 facts.FiberG,
        ProteinG:               facts.ProteinG,
        SaltG:                  facts.SaltG,
    }, nil
}
