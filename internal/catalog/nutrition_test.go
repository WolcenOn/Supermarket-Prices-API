package catalog

import (
    "encoding/json"
    "testing"
)

func TestProductNutritionProportionalCalculationReady(t *testing.T) {
    basis100 := 100.0

    ready := ProductNutrition{
        Source:      "dia_product_page",
        BasisAmount: &basis100,
        BasisUnit:   "g",
    }
    if !ready.ProportionalCalculationReady() {
        t.Fatal("expected declared 100 g basis to be calculation-ready")
    }

    missingBasis := ProductNutrition{
        Source: "dia_product_page",
    }
    if missingBasis.ProportionalCalculationReady() {
        t.Fatal("nutrition without a declared basis must not be calculation-ready")
    }

    unsupportedBasis := ProductNutrition{
        Source:      "dia_product_page",
        BasisAmount: &basis100,
        BasisUnit:   "serving",
    }
    if unsupportedBasis.ProportionalCalculationReady() {
        t.Fatal("unsupported basis unit must not be calculation-ready")
    }
}

func TestProductNutritionJSONExposesDerivedCalculationReadiness(t *testing.T) {
    basis100 := 100.0
    cases := []struct {
        name     string
        item     ProductNutrition
        expected bool
    }{
        {
            name: "declared 100 g basis",
            item: ProductNutrition{
                ExternalID:  "21415",
                Source:      "dia_product_page",
                BasisAmount: &basis100,
                BasisUnit:   "g",
            },
            expected: true,
        },
        {
            name: "DIA nutrition without published basis",
            item: ProductNutrition{
                ExternalID: "267141",
                Source:     "dia_product_page",
            },
            expected: false,
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            encoded, err := json.Marshal(tc.item)
            if err != nil {
                t.Fatal(err)
            }

            var body struct {
                ExternalID                   string `json:"externalId"`
                ProportionalCalculationReady bool   `json:"proportionalCalculationReady"`
            }
            if err := json.Unmarshal(encoded, &body); err != nil {
                t.Fatal(err)
            }
            if body.ExternalID != tc.item.ExternalID {
                t.Fatalf("unexpected external id %q", body.ExternalID)
            }
            if body.ProportionalCalculationReady != tc.expected {
                t.Fatalf("expected readiness %v, got %v: %s", tc.expected, body.ProportionalCalculationReady, encoded)
            }
        })
    }
}
