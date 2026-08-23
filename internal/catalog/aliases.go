package catalog

import "context"

type CanonicalIngredientAlias struct {
    ID                    int64   `json:"id"`
    CanonicalIngredientID string  `json:"canonicalIngredientId"`
    Alias                 string  `json:"alias"`
    NormalizedAlias       string  `json:"normalizedAlias"`
    Status                string  `json:"status"`
    Confidence            float64 `json:"confidence,omitempty"`
    DecisionSource        string  `json:"decisionSource,omitempty"`
    VerificationNote      string  `json:"verificationNote,omitempty"`
}

type CanonicalResolutionCandidate struct {
    Ingredient CanonicalIngredient       `json:"ingredient"`
    Alias      *CanonicalIngredientAlias `json:"alias,omitempty"`
    MatchType  string                    `json:"matchType"`
    Confidence float64                   `json:"confidence"`
}

type CanonicalResolution struct {
    Query           string                         `json:"query"`
    NormalizedQuery string                         `json:"normalizedQuery"`
    Status          string                         `json:"status"`
    Candidates      []CanonicalResolutionCandidate `json:"candidates"`
}

type CanonicalResolverStore interface {
    ResolveCanonicalIngredient(ctx context.Context, query string) (CanonicalResolution, error)
}
