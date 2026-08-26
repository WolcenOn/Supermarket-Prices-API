package canonicalcoverage

import (
	"context"
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type fakeResolver map[string]catalog.CanonicalResolution

func (f fakeResolver) ResolveCanonicalIngredient(_ context.Context, query string) (catalog.CanonicalResolution, error) {
	return f[query], nil
}

func TestAnalyzeClassifiesCoverageConservatively(t *testing.T) {
	resolver := fakeResolver{
		"Calabaza": {
			Query: "Calabaza", NormalizedQuery: "calabaza", Status: "verified",
			Candidates: []catalog.CanonicalResolutionCandidate{{
				Ingredient: catalog.CanonicalIngredient{ID: "calabaza", Name: "Calabaza"},
				MatchType: "canonical_name", Confidence: 1,
			}},
		},
		"Espinacas": {
			Query: "Espinacas", NormalizedQuery: "espinacas", Status: "verified",
			Candidates: []catalog.CanonicalResolutionCandidate{{
				Ingredient: catalog.CanonicalIngredient{ID: "espinaca", Name: "Espinaca"},
				MatchType: "alias", Confidence: 0.99,
			}},
		},
		"Arroz blanco": {
			Query: "Arroz blanco", NormalizedQuery: "arroz blanco", Status: "suggested",
			Candidates: []catalog.CanonicalResolutionCandidate{
				{Ingredient: catalog.CanonicalIngredient{ID: "arroz_redondo", Name: "Arroz redondo"}, MatchType: "alias", Confidence: 0.8},
				{Ingredient: catalog.CanonicalIngredient{ID: "arroz_extra", Name: "Arroz extra"}, MatchType: "alias", Confidence: 0.75},
			},
		},
		"Pechuga de pollo": {Query: "Pechuga de pollo", NormalizedQuery: "pechuga de pollo", Status: "unresolved"},
	}

	report, err := Analyze(context.Background(), resolver, []Input{
		{Name: "Calabaza", Count: 3, Source: "pack-a"},
		{Name: "Espinacas", Count: 2, Source: "pack-b"},
		{Name: "Arroz blanco", Count: 4, Source: "pack-c"},
		{Name: "Pechuga de pollo", Count: 5, Source: "pack-d"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if report.Summary.UniqueIngredients != 4 || report.Summary.TotalOccurrences != 14 {
		t.Fatalf("unexpected totals: %+v", report.Summary)
	}
	if report.Summary.ResolvedUnique != 2 || report.Summary.ResolvedOccurrences != 5 {
		t.Fatalf("unexpected resolved totals: %+v", report.Summary)
	}
	if report.Summary.CanonicalExact != 1 || report.Summary.VerifiedAlias != 1 || report.Summary.Ambiguous != 1 || report.Summary.Unresolved != 1 {
		t.Fatalf("unexpected statuses: %+v", report.Summary)
	}
	if report.Items[0].Name != "Pechuga de pollo" || report.Items[0].Status != StatusUnresolved {
		t.Fatalf("highest-priority unresolved item should be first: %+v", report.Items[0])
	}
}

func TestAnalyzeDoesNotTreatSingleSuggestedAliasAsResolved(t *testing.T) {
	resolver := fakeResolver{
		"Arroz blanco": {
			Query: "Arroz blanco", NormalizedQuery: "arroz blanco", Status: "suggested",
			Candidates: []catalog.CanonicalResolutionCandidate{{
				Ingredient: catalog.CanonicalIngredient{ID: "arroz_redondo", Name: "Arroz redondo"},
				MatchType: "alias", Confidence: 0.9,
			}},
		},
	}

	report, err := Analyze(context.Background(), resolver, []Input{{Name: "Arroz blanco", Count: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.ResolvedOccurrences != 0 || report.Summary.SuggestedAlias != 1 {
		t.Fatalf("suggestions must require review: %+v", report.Summary)
	}
}
