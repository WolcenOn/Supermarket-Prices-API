package catalogscan

import (
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestAnalyzeAutoMatchCandidate(t *testing.T) {
	product := catalog.Product{
		SupermarketID:        "dia",
		ExternalID:           "110",
		Name:                 "Tomate ensalada granel 1 Kg aprox.",
		SourceCategoryID:     "L2023",
		ItemType:             "food_ingredient",
		NormalizedCategory:   "food.produce.vegetable",
		RecipeCompatible:     true,
		ClassificationStatus: "classified",
		Price:                1.99,
	}

	analysis := Analyze(product)
	if analysis.Decision != DecisionAutoMatchCandidate {
		t.Fatalf("decision = %q, want %q", analysis.Decision, DecisionAutoMatchCandidate)
	}
	if len(analysis.MatchCandidates) != 1 {
		t.Fatalf("match candidates = %d, want 1", len(analysis.MatchCandidates))
	}
	if got := analysis.MatchCandidates[0].CanonicalIngredientID; got != "tomate" {
		t.Fatalf("canonical = %q, want tomate", got)
	}
}

func TestAnalyzeNonRecipeDoesNotSuggestMatch(t *testing.T) {
	product := catalog.Product{
		SupermarketID:        "dia",
		ExternalID:           "prepared",
		Name:                 "Ensalada gourmet Dia Vegecampo 150 g",
		ClassificationStatus: "classified",
		RecipeCompatible:     false,
		Price:                1.50,
	}

	analysis := Analyze(product)
	if analysis.Decision != DecisionNonRecipe {
		t.Fatalf("decision = %q, want %q", analysis.Decision, DecisionNonRecipe)
	}
	if len(analysis.MatchCandidates) != 0 {
		t.Fatalf("match candidates = %d, want 0", len(analysis.MatchCandidates))
	}
}

func TestAnalyzeReviewWhenRecipeProductHasNoMatch(t *testing.T) {
	product := catalog.Product{
		SupermarketID:        "dia",
		ExternalID:           "190386",
		Name:                 "Yuca unidad 1 Kg aprox.",
		SourceCategoryID:     "L2028",
		ItemType:             "food_ingredient",
		NormalizedCategory:   "food.produce.vegetable",
		RecipeCompatible:     true,
		ClassificationStatus: "classified",
		Price:                2.10,
	}

	analysis := Analyze(product)
	if analysis.Decision != DecisionReview {
		t.Fatalf("decision = %q, want %q", analysis.Decision, DecisionReview)
	}
	if len(analysis.MatchCandidates) != 0 {
		t.Fatalf("match candidates = %d, want 0", len(analysis.MatchCandidates))
	}
}

func TestAnalyzeInvalidIdentity(t *testing.T) {
	analysis := Analyze(catalog.Product{Name: "Producto sin SKU", SupermarketID: "dia"})
	if analysis.Decision != DecisionInvalid {
		t.Fatalf("decision = %q, want %q", analysis.Decision, DecisionInvalid)
	}
	if len(analysis.Issues) == 0 || analysis.Issues[0].Code != "missing_external_id" {
		t.Fatalf("issues = %#v, want missing_external_id", analysis.Issues)
	}
}
