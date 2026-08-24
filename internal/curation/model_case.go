package curation

import (
	"context"
	"strings"
)

// ModelCase is the closed, deterministic context given to a curation model.
// The model may judge this candidate or abstain, but it may not redirect the
// proposal to a different alias, canonical ingredient, or evidence source.
type ModelCase struct {
	Alias                 string            `json:"alias"`
	CanonicalIngredientID string            `json:"canonicalIngredientId"`
	CanonicalName         string            `json:"canonicalName"`
	Products              []ProductEvidence `json:"products"`
}

// VerifyModelProposal first binds an untrusted model response to the exact
// input case and only then runs the normal deterministic curation policy.
func VerifyModelProposal(ctx context.Context, modelCase ModelCase, proposal Proposal, lookup Lookup) (Verdict, error) {
	vetoes := modelBindingVetoes(modelCase, proposal)
	if len(vetoes) > 0 {
		return Verdict{
			Status:            VerdictRejected,
			PolicyVersion:     PolicyVersionV1,
			EligibleToSuggest: false,
			Vetoes:            vetoes,
			Warnings:          []string{},
		}, nil
	}
	return Verify(ctx, proposal, lookup)
}

func modelBindingVetoes(modelCase ModelCase, proposal Proposal) []string {
	if strings.TrimSpace(strings.ToLower(proposal.Action)) == ActionAbstain {
		return nil
	}

	vetoes := make([]string, 0)
	if strings.TrimSpace(proposal.Alias) != strings.TrimSpace(modelCase.Alias) {
		vetoes = append(vetoes, "model_changed_alias")
	}
	if strings.TrimSpace(proposal.CanonicalIngredientID) != strings.TrimSpace(modelCase.CanonicalIngredientID) {
		vetoes = append(vetoes, "model_changed_canonical_ingredient")
	}

	allowedProducts := make(map[string]struct{}, len(modelCase.Products))
	for _, product := range modelCase.Products {
		id := strings.TrimSpace(product.ID)
		if id != "" {
			allowedProducts[id] = struct{}{}
		}
	}
	for _, evidence := range proposal.Evidence {
		if strings.TrimSpace(evidence.Type) != "supermarket_product" {
			vetoes = append(vetoes, "model_used_unsupplied_evidence_type:"+strings.TrimSpace(evidence.Type))
			continue
		}
		productID := strings.TrimSpace(evidence.SupermarketProductID)
		if _, ok := allowedProducts[productID]; !ok {
			vetoes = append(vetoes, "model_used_unsupplied_product:"+productID)
		}
	}
	return vetoes
}
