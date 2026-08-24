package curation

import (
	"context"
	"testing"
)

func modelCaseFixture() ModelCase {
	return ModelCase{
		Alias:                 "Arroz de grano redondo",
		CanonicalIngredientID: "arroz_redondo",
		CanonicalName:         "Arroz redondo",
		Products: []ProductEvidence{{
			ID:                   "product-1",
			Name:                 "Arroz redondo 1 kg",
			ItemType:             "food_ingredient",
			NormalizedCategory:   "food.pantry.cereal.rice",
			RecipeCompatible:     true,
			ClassificationStatus: "classified",
		}},
	}
}

func modelProposalFixture() Proposal {
	return Proposal{
		SchemaVersion:         SchemaVersionV1,
		PolicyVersion:         PolicyVersionV1,
		Action:                ActionProposeAlias,
		Alias:                 "Arroz de grano redondo",
		CanonicalIngredientID: "arroz_redondo",
		Confidence:            0.97,
		Reasons:               []string{"misma variedad de arroz crudo"},
		Evidence: []Evidence{{
			Type:                 "supermarket_product",
			SupermarketProductID: "product-1",
		}},
	}
}

func TestVerifyModelProposalRejectsChangedAlias(t *testing.T) {
	proposal := modelProposalFixture()
	proposal.Alias = "Arroz tres delicias"

	verdict, err := VerifyModelProposal(context.Background(), modelCaseFixture(), proposal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictRejected || verdict.EligibleToSuggest {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
	if len(verdict.Vetoes) != 1 || verdict.Vetoes[0] != "model_changed_alias" {
		t.Fatalf("unexpected vetoes: %#v", verdict.Vetoes)
	}
}

func TestVerifyModelProposalRejectsChangedCanonicalIngredient(t *testing.T) {
	proposal := modelProposalFixture()
	proposal.CanonicalIngredientID = "arroz_basmati"

	verdict, err := VerifyModelProposal(context.Background(), modelCaseFixture(), proposal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(verdict.Vetoes) != 1 || verdict.Vetoes[0] != "model_changed_canonical_ingredient" {
		t.Fatalf("unexpected vetoes: %#v", verdict.Vetoes)
	}
}

func TestVerifyModelProposalRejectsInventedProductEvidence(t *testing.T) {
	proposal := modelProposalFixture()
	proposal.Evidence[0].SupermarketProductID = "invented-product"

	verdict, err := VerifyModelProposal(context.Background(), modelCaseFixture(), proposal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(verdict.Vetoes) != 1 || verdict.Vetoes[0] != "model_used_unsupplied_product:invented-product" {
		t.Fatalf("unexpected vetoes: %#v", verdict.Vetoes)
	}
}

func TestVerifyModelProposalAllowsAbstentionWithoutRebinding(t *testing.T) {
	proposal := Proposal{
		SchemaVersion: SchemaVersionV1,
		PolicyVersion: PolicyVersionV1,
		Action:        ActionAbstain,
	}

	verdict, err := VerifyModelProposal(context.Background(), modelCaseFixture(), proposal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Status != VerdictNeedsReview || verdict.EligibleToSuggest {
		t.Fatalf("unexpected verdict: %+v", verdict)
	}
}
