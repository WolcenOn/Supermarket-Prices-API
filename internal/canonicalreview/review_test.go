package canonicalreview

import (
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalgaps"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalsemantics"
)

func TestBuildGroupsEquivalentMissingConcepts(t *testing.T) {
	input := canonicalgaps.Report{
		ProductsTotal: 100,
		ProductsUsed: 80,
		Items: []canonicalgaps.Item{
			{Family: "pollo", Role: canonicalsemantics.RoleIngredientCandidate, Pattern: "pechuga pollo", SuggestedConceptID: "pechuga_pollo_nueva", SupportCount: 5, EvidenceScore: 0.90, Status: canonicalgaps.StatusMissing},
			{Family: "pollo", Role: canonicalsemantics.RoleIngredientCandidate, Pattern: "pollo pechuga", SuggestedConceptID: "pollo_pechuga_nueva", SupportCount: 4, EvidenceScore: 0.88, Status: canonicalgaps.StatusMissing},
		},
	}

	report := Build(input)
	if report.Summary.MissingItems != 2 {
		t.Fatalf("missing items = %d, want 2", report.Summary.MissingItems)
	}
	if report.Summary.GroupedCandidates != 1 {
		t.Fatalf("grouped candidates = %d, want 1", report.Summary.GroupedCandidates)
	}
	candidate := report.Candidates[0]
	if candidate.SupportTotal != 9 {
		t.Fatalf("support total = %d, want 9", candidate.SupportTotal)
	}
	if candidate.Priority != PriorityHigh {
		t.Fatalf("priority = %s, want %s", candidate.Priority, PriorityHigh)
	}
	if len(candidate.Patterns) != 2 {
		t.Fatalf("patterns = %d, want 2", len(candidate.Patterns))
	}
}

func TestBuildIgnoresAlreadyCoveredItems(t *testing.T) {
	input := canonicalgaps.Report{Items: []canonicalgaps.Item{
		{Family: "arroz", Pattern: "arroz basmati", Status: canonicalgaps.StatusCanonicalExact, SupportCount: 12, EvidenceScore: 0.95},
		{Family: "arroz", Role: canonicalsemantics.RoleIngredientCandidate, Pattern: "arroz bomba", SuggestedConceptID: "arroz_bomba", Status: canonicalgaps.StatusMissing, SupportCount: 3, EvidenceScore: 0.79},
	}}
	report := Build(input)
	if len(report.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(report.Candidates))
	}
	if report.Candidates[0].SuggestedConceptID != "arroz_bomba" {
		t.Fatalf("concept = %q, want arroz_bomba", report.Candidates[0].SuggestedConceptID)
	}
}

func TestBuildKeepsFamiliesSeparate(t *testing.T) {
	input := canonicalgaps.Report{Items: []canonicalgaps.Item{
		{Family: "pollo", Role: canonicalsemantics.RoleIngredientCandidate, Pattern: "pechuga", SuggestedConceptID: "pechuga", Status: canonicalgaps.StatusMissing, SupportCount: 3, EvidenceScore: 0.8},
		{Family: "pavo", Role: canonicalsemantics.RoleIngredientCandidate, Pattern: "pechuga", SuggestedConceptID: "pechuga", Status: canonicalgaps.StatusMissing, SupportCount: 3, EvidenceScore: 0.8},
	}}
	report := Build(input)
	if len(report.Candidates) != 2 {
		t.Fatalf("candidates = %d, want 2", len(report.Candidates))
	}
}
