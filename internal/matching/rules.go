package matching

import (
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type Match struct {
    CanonicalIngredientID string
    SupermarketID         string
    ExternalID            string
    Score                 float64
    Status                string
    Source                string
}

type rule struct {
    ingredientID string
    phrase       string
    score        float64
}

var riceRules = []rule{
    {ingredientID: "arroz_redondo", phrase: "arroz redondo", score: 0.99},
    {ingredientID: "arroz_extra", phrase: "arroz extra", score: 0.99},
    {ingredientID: "arroz_vaporizado", phrase: "arroz vaporizado", score: 0.99},
    {ingredientID: "arroz_basmati", phrase: "arroz basmati", score: 0.99},
    {ingredientID: "arroz_integral", phrase: "arroz integral", score: 0.99},
}

var preparedRiceTerms = []string{
    "tres delicias",
    "al punto",
    "vasos de arroz",
    "vaso de arroz",
    "marisco",
    "secreto iberico",
    "secreto ibérico",
    "risotto",
    "paella preparada",
}

// Suggest returns only high-confidence, recipe-level equivalences. Ambiguous
// products intentionally remain unmatched for later review instead of forcing a
// potentially wrong canonical ingredient.
func Suggest(product catalog.Product) []Match {
    name := normalize(product.Name)
    if name == "" || isPreparedRice(name) {
        return nil
    }

    out := make([]Match, 0, 1)
    for _, candidate := range riceRules {
        if strings.Contains(name, candidate.phrase) {
            out = append(out, Match{
                CanonicalIngredientID: candidate.ingredientID,
                SupermarketID: product.SupermarketID,
                ExternalID: product.ExternalID,
                Score: candidate.score,
                Status: "automatic",
                Source: "rules:v1",
            })
            break
        }
    }
    return out
}

func isPreparedRice(name string) bool {
    for _, term := range preparedRiceTerms {
        if strings.Contains(name, term) {
            return true
        }
    }
    return false
}

func normalize(value string) string {
    return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}
