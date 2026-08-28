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

func TestAnalyzeUsesSpecificCanonicalCandidates(t *testing.T) {
	cases := []struct {
		categoryID string
		name       string
		want       string
	}{
		{"L2022", "Cebolla morada malla 500 g", "cebolla_morada"},
		{"L2023", "Pimiento rojo unidad 350 g aprox.", "pimiento_rojo"},
		{"L2023", "Tomates cherry bandeja 400 g", "tomate_cherry"},
		{"L2027", "Canónigos Dia Vegecampo 70 g", "canonigos"},
	}
	for _, tc := range cases {
		product := catalog.Product{
			SupermarketID:        "dia",
			ExternalID:           "sku",
			Name:                 tc.name,
			SourceCategoryID:     tc.categoryID,
			ItemType:             "food_ingredient",
			NormalizedCategory:   "food.produce.vegetable",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
			Price:                1,
		}
		analysis := Analyze(product)
		if analysis.Decision != DecisionAutoMatchCandidate || len(analysis.MatchCandidates) != 1 || analysis.MatchCandidates[0].CanonicalIngredientID != tc.want {
			t.Fatalf("%q: got %#v, want auto candidate %s", tc.name, analysis, tc.want)
		}
	}
}

func TestAnalyzeMatchesMushroomClassification(t *testing.T) {
	product := catalog.Product{
		SupermarketID:        "dia",
		ExternalID:           "271527",
		Name:                 "Champiñón entero bandeja 250 g",
		SourceCategoryID:     "L2029",
		ItemType:             "food_ingredient",
		NormalizedCategory:   "food.produce.mushroom",
		RecipeCompatible:     true,
		ClassificationStatus: "classified",
		Price:                1.39,
	}
	analysis := Analyze(product)
	if analysis.Decision != DecisionAutoMatchCandidate || len(analysis.MatchCandidates) != 1 || analysis.MatchCandidates[0].CanonicalIngredientID != "champinon" {
		t.Fatalf("analysis = %#v, want champinon auto candidate", analysis)
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

func TestAnalyzeNoCanonicalForSimpleProduce(t *testing.T) {
	cases := []string{
		"Yuca unidad 1 Kg aprox.",
		"Batata unidad 800 g aprox.",
		"Repollo liso unidad 2 Kg aprox.",
		"Cebolla tierna unidad",
	}
	for _, name := range cases {
		product := catalog.Product{
			SupermarketID:        "dia",
			ExternalID:           "sku",
			Name:                 name,
			SourceCategoryID:     "L2028",
			ItemType:             "food_ingredient",
			NormalizedCategory:   "food.produce.vegetable",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
			Price:                2.10,
		}
		if name == "Repollo liso unidad 2 Kg aprox." {
			product.SourceCategoryID = "L2024"
		}
		if name == "Cebolla tierna unidad" {
			product.SourceCategoryID = "L2022"
		}
		analysis := Analyze(product)
		if analysis.Decision != DecisionNoCanonical {
			t.Fatalf("%q decision = %q, want %q", name, analysis.Decision, DecisionNoCanonical)
		}
	}
}

func TestAnalyzeKeepsPreparedProduceInReview(t *testing.T) {
	cases := []struct {
		categoryID string
		name       string
	}{
		{"L2024", "Verduras para microondas Dia Vegecampo 300 g"},
		{"L2024", "Brócoli para microondas Florette 225 g"},
		{"L2027", "Ensalada 4 estaciones Dia Vegecampo 200 g"},
	}
	for _, tc := range cases {
		product := catalog.Product{
			SupermarketID:        "dia",
			ExternalID:           "sku",
			Name:                 tc.name,
			SourceCategoryID:     tc.categoryID,
			ItemType:             "food_ingredient",
			NormalizedCategory:   "food.produce.vegetable",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
			Price:                1.99,
		}
		analysis := Analyze(product)
		if analysis.Decision != DecisionReview {
			t.Fatalf("%q decision = %q, want %q", tc.name, analysis.Decision, DecisionReview)
		}
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
