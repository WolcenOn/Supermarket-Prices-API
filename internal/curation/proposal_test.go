package curation

import (
	"context"
	"testing"
)

type fakeProductLookup map[string]ProductEvidence

func (f fakeProductLookup) CurationProductEvidence(_ context.Context, productID string) (ProductEvidence, bool, error) {
	product, ok := f[productID]
	return product, ok, nil
}

func baseProposal() Proposal {
	return Proposal{
		SchemaVersion:         SchemaVersionV1,
		PolicyVersion:         PolicyVersionV1,
		Action:                ActionProposeAlias,
		Alias:                 "Arroz de grano redondo",
		CanonicalIngredientID: "arroz_redondo",
		Confidence:            0.97,
		Reasons:               []string{"mismo ingrediente base", "el modificador grano no cambia el concepto"},
		Evidence: []Evidence{{
			Type:                 "supermarket_product",
			SupermarketProductID: "raw-rice",
		}},
	}
}

func TestVerifyAcceptsHighConfidenceCompatibleProposalForSuggestion(t *testing.T) {
	proposal := baseProposal()
	lookup := fakeProductLookup{
		"raw-rice": {
			ID:                   "raw-rice",
			Name:                 "Arroz redondo 1 kg",
			ItemType:             "food_ingredient",
			NormalizedCategory:   "food.pantry.cereal.rice",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
			ClassificationSource: "rules:v1",
		},
	}

	verdict, err := Verify(context.Background(), proposal, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictAccepted || !verdict.EligibleToSuggest {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}

func TestVerifyVetoesPreparedFoodEvenWhenAgentIsConfident(t *testing.T) {
	proposal := baseProposal()
	proposal.Alias = "Arroz tres delicias"
	proposal.Confidence = 0.99
	proposal.Evidence[0].SupermarketProductID = "prepared-rice"
	lookup := fakeProductLookup{
		"prepared-rice": {
			ID:                   "prepared-rice",
			Name:                 "Arroz tres delicias",
			ItemType:             "prepared_food",
			NormalizedCategory:   "food.prepared",
			RecipeCompatible:     false,
			ClassificationStatus: "classified",
			ClassificationSource: "rules:v1",
		},
	}

	verdict, err := Verify(context.Background(), proposal, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictRejected || verdict.EligibleToSuggest {
		t.Fatalf("expected deterministic veto, got %+v", verdict)
	}
	if len(verdict.Vetoes) != 1 || verdict.Vetoes[0] != "product_not_recipe_compatible:prepared-rice" {
		t.Fatalf("unexpected vetoes: %#v", verdict.Vetoes)
	}
}

func TestVerifySendsAmbiguousOrWeakProposalToReview(t *testing.T) {
	proposal := baseProposal()
	proposal.Confidence = 0.82
	lookup := fakeProductLookup{
		"raw-rice": {
			ID:                   "raw-rice",
			ItemType:             "food_ingredient",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
		},
	}

	verdict, err := Verify(context.Background(), proposal, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictNeedsReview || verdict.EligibleToSuggest {
		t.Fatalf("expected needs_review, got %+v", verdict)
	}
}

func TestVerifyDoesNotAutoAcceptAgentReportedConflict(t *testing.T) {
	proposal := baseProposal()
	proposal.Conflicts = []string{"podría confundirse con arroz largo"}
	lookup := fakeProductLookup{
		"raw-rice": {
			ID:                   "raw-rice",
			ItemType:             "food_ingredient",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
		},
	}

	verdict, err := Verify(context.Background(), proposal, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictNeedsReview || verdict.EligibleToSuggest {
		t.Fatalf("expected needs_review, got %+v", verdict)
	}
}

func TestVerifyTreatsAgentAbstentionAsNeedsReview(t *testing.T) {
	proposal := Proposal{
		SchemaVersion: SchemaVersionV1,
		PolicyVersion: PolicyVersionV1,
		Action:        ActionAbstain,
	}
	verdict, err := Verify(context.Background(), proposal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictNeedsReview || verdict.EligibleToSuggest {
		t.Fatalf("expected needs_review, got %+v", verdict)
	}
}
