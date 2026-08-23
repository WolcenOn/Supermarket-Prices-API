package curation

import (
	"context"
	"fmt"
	"strings"
)

const (
	SchemaVersionV1 = "canonical-curation:v1"
	PolicyVersionV1 = "canonical-curation-policy:v1"

	ActionProposeAlias = "propose_alias"
	ActionAbstain      = "abstain"

	VerdictAccepted   = "accepted_for_suggestion"
	VerdictNeedsReview = "needs_review"
	VerdictRejected   = "rejected"
)

type Proposal struct {
	SchemaVersion         string     `json:"schemaVersion"`
	PolicyVersion         string     `json:"policyVersion"`
	Action                string     `json:"action"`
	Alias                 string     `json:"alias,omitempty"`
	CanonicalIngredientID string     `json:"canonicalIngredientId,omitempty"`
	Confidence            float64    `json:"confidence"`
	Reasons               []string   `json:"reasons,omitempty"`
	Conflicts             []string   `json:"conflicts,omitempty"`
	Evidence              []Evidence `json:"evidence,omitempty"`
	Agent                  AgentInfo  `json:"agent,omitempty"`
}

type AgentInfo struct {
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
	RunID    string `json:"runId,omitempty"`
}

type Evidence struct {
	Type                 string `json:"type"`
	SupermarketProductID string `json:"supermarketProductId,omitempty"`
	SourceRef            string `json:"sourceRef,omitempty"`
	SourceText           string `json:"sourceText,omitempty"`
}

type ProductEvidence struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	ItemType             string `json:"itemType"`
	NormalizedCategory   string `json:"normalizedCategory"`
	RecipeCompatible     bool   `json:"recipeCompatible"`
	ClassificationStatus string `json:"classificationStatus"`
	ClassificationSource string `json:"classificationSource"`
}

type ProductEvidenceLookup interface {
	CurationProductEvidence(ctx context.Context, productID string) (ProductEvidence, bool, error)
}

type Verdict struct {
	Status            string            `json:"status"`
	PolicyVersion     string            `json:"policyVersion"`
	EligibleToSuggest bool              `json:"eligibleToSuggest"`
	Vetoes            []string          `json:"vetoes"`
	Warnings          []string          `json:"warnings"`
	CheckedProducts   []ProductEvidence `json:"checkedProducts,omitempty"`
}

func Verify(ctx context.Context, proposal Proposal, products ProductEvidenceLookup) (Verdict, error) {
	verdict := Verdict{
		Status:        VerdictRejected,
		PolicyVersion: PolicyVersionV1,
		Vetoes:        []string{},
		Warnings:      []string{},
	}

	proposal.SchemaVersion = strings.TrimSpace(proposal.SchemaVersion)
	proposal.PolicyVersion = strings.TrimSpace(proposal.PolicyVersion)
	proposal.Action = strings.TrimSpace(strings.ToLower(proposal.Action))
	proposal.Alias = strings.TrimSpace(proposal.Alias)
	proposal.CanonicalIngredientID = strings.TrimSpace(proposal.CanonicalIngredientID)

	if proposal.SchemaVersion != SchemaVersionV1 {
		verdict.Vetoes = append(verdict.Vetoes, "unsupported_schema_version")
	}
	if proposal.PolicyVersion != PolicyVersionV1 {
		verdict.Vetoes = append(verdict.Vetoes, "unsupported_policy_version")
	}

	if proposal.Action == ActionAbstain {
		verdict.Status = VerdictNeedsReview
		verdict.Warnings = append(verdict.Warnings, "agent_abstained")
		return verdict, nil
	}
	if proposal.Action != ActionProposeAlias {
		verdict.Vetoes = append(verdict.Vetoes, "unsupported_action")
	}
	if proposal.Alias == "" {
		verdict.Vetoes = append(verdict.Vetoes, "missing_alias")
	}
	if proposal.CanonicalIngredientID == "" {
		verdict.Vetoes = append(verdict.Vetoes, "missing_canonical_ingredient_id")
	}
	if proposal.Confidence <= 0 || proposal.Confidence > 1 {
		verdict.Vetoes = append(verdict.Vetoes, "invalid_confidence")
	}
	if len(nonEmpty(proposal.Reasons)) == 0 {
		verdict.Warnings = append(verdict.Warnings, "missing_reasons")
	}
	if len(nonEmpty(proposal.Conflicts)) > 0 {
		verdict.Warnings = append(verdict.Warnings, "agent_reported_conflicts")
	}
	if len(proposal.Evidence) == 0 {
		verdict.Warnings = append(verdict.Warnings, "missing_evidence")
	}

	if len(verdict.Vetoes) > 0 {
		return verdict, nil
	}

	for _, evidence := range proposal.Evidence {
		evidence.Type = strings.TrimSpace(evidence.Type)
		productID := strings.TrimSpace(evidence.SupermarketProductID)
		if evidence.Type != "supermarket_product" {
			continue
		}
		if productID == "" {
			verdict.Vetoes = append(verdict.Vetoes, "supermarket_product_evidence_missing_product_id")
			continue
		}
		if products == nil {
			return Verdict{}, fmt.Errorf("curation: product evidence lookup is required")
		}
		product, found, err := products.CurationProductEvidence(ctx, productID)
		if err != nil {
			return Verdict{}, fmt.Errorf("curation: load product evidence %s: %w", productID, err)
		}
		if !found {
			verdict.Warnings = append(verdict.Warnings, "product_evidence_not_found:"+productID)
			continue
		}
		verdict.CheckedProducts = append(verdict.CheckedProducts, product)

		if product.ClassificationStatus != "classified" {
			verdict.Warnings = append(verdict.Warnings, "product_not_classified:"+productID)
			continue
		}
		if product.ItemType != "food_ingredient" || !product.RecipeCompatible {
			verdict.Vetoes = append(verdict.Vetoes, "product_not_recipe_compatible:"+productID)
		}
	}

	if len(verdict.Vetoes) > 0 {
		verdict.Status = VerdictRejected
		return verdict, nil
	}

	if proposal.Confidence < 0.90 || len(nonEmpty(proposal.Conflicts)) > 0 || len(proposal.Evidence) == 0 || len(nonEmpty(proposal.Reasons)) == 0 {
		verdict.Status = VerdictNeedsReview
		return verdict, nil
	}
	for _, warning := range verdict.Warnings {
		if strings.HasPrefix(warning, "product_evidence_not_found:") || strings.HasPrefix(warning, "product_not_classified:") {
			verdict.Status = VerdictNeedsReview
			return verdict, nil
		}
	}

	verdict.Status = VerdictAccepted
	verdict.EligibleToSuggest = true
	return verdict, nil
}

func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}
