package canonicalgaps

import (
	"context"
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalruleproposal"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalsemantics"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type fakeStore struct {
	ingredients []catalog.CanonicalIngredient
	resolutions map[string]catalog.CanonicalResolution
}

func (f fakeStore) Ingredients(context.Context) ([]catalog.CanonicalIngredient, error) {
	return f.ingredients, nil
}

func (f fakeStore) ResolveCanonicalIngredient(_ context.Context, query string) (catalog.CanonicalResolution, error) {
	if resolution, ok := f.resolutions[query]; ok {
		return resolution, nil
	}
	return catalog.CanonicalResolution{Query: query, Status: "unresolved"}, nil
}

func TestAnalyzeFindsExactSemanticAndMissingConcepts(t *testing.T) {
	store := fakeStore{
		ingredients: []catalog.CanonicalIngredient{
			{ID: "pechuga_pollo", Name: "Pechuga de pollo"},
			{ID: "arroz_basmati", Name: "Arroz basmati"},
		},
		resolutions: map[string]catalog.CanonicalResolution{
			"arroz_basmati": {
				Status: "verified",
				Candidates: []catalog.CanonicalResolutionCandidate{{
					Ingredient: catalog.CanonicalIngredient{ID: "arroz_basmati", Name: "Arroz basmati"},
					MatchType: "canonical_name", Confidence: 1,
				}},
			},
		},
	}
	input := canonicalsemantics.Report{Families: []canonicalsemantics.FamilyReport{
		{Family: "arroz", Proposals: []canonicalsemantics.AuditedProposal{{
			Proposal: canonicalruleproposal.Proposal{Pattern: "arroz basmati", SuggestedConceptID: "arroz_basmati", SupportCount: 8, EvidenceScore: .9},
			Role: canonicalsemantics.RoleIngredientCandidate,
		}}},
		{Family: "pollo", Proposals: []canonicalsemantics.AuditedProposal{{
			Proposal: canonicalruleproposal.Proposal{Pattern: "pollo pechuga", SuggestedConceptID: "pollo_pechuga", SupportCount: 6, EvidenceScore: .88},
			Role: canonicalsemantics.RoleIngredientCandidate,
		}}},
		{Family: "tomate", Proposals: []canonicalsemantics.AuditedProposal{{
			Proposal: canonicalruleproposal.Proposal{Pattern: "tomate seco", SuggestedConceptID: "tomate_seco", SupportCount: 4, EvidenceScore: .8},
			Role: canonicalsemantics.RoleProcessedIngredient,
		}}},
	}}

	report, err := Analyze(context.Background(), store, input)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Candidates != 3 || report.Summary.CanonicalExact != 1 || report.Summary.SemanticEquivalent != 1 || report.Summary.Missing != 1 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	var semantic Item
	for _, item := range report.Items {
		if item.SuggestedConceptID == "pollo_pechuga" {
			semantic = item
		}
	}
	if semantic.Status != StatusSemanticEquivalent || semantic.CanonicalID != "pechuga_pollo" {
		t.Fatalf("semantic match = %+v", semantic)
	}
}

func TestAnalyzeIgnoresPreparedAndContextRoles(t *testing.T) {
	input := canonicalsemantics.Report{Families: []canonicalsemantics.FamilyReport{{
		Family: "jamon",
		Proposals: []canonicalsemantics.AuditedProposal{
			{Proposal: canonicalruleproposal.Proposal{Pattern: "pizza jamon", SuggestedConceptID: "pizza_jamon"}, Role: canonicalsemantics.RolePreparedContext},
			{Proposal: canonicalruleproposal.Proposal{Pattern: "sabor jamon", SuggestedConceptID: "sabor_jamon"}, Role: canonicalsemantics.RoleContextOnly},
		},
	}}}
	report, err := Analyze(context.Background(), fakeStore{}, input)
	if err != nil {
		t.Fatal(err)
	}
	if report.Summary.Candidates != 0 || len(report.Items) != 0 {
		t.Fatalf("expected no catalog candidates, got %+v", report)
	}
}
