package main

import (
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

func validConfig() config {
	return config{
		CanonicalID:    "arroz_redondo",
		Alias:          "Arroz de grano redondo",
		Confidence:     0.95,
		DecisionSource: "manual:cli",
		EvidenceType:   "manual",
		SourceText:     "reviewed semantic equivalence",
	}
}

func TestValidateConfigAcceptsExplicitEvidence(t *testing.T) {
	if err := validateConfig(validConfig()); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}
}

func TestValidateConfigRequiresEvidence(t *testing.T) {
	cfg := validConfig()
	cfg.SourceText = ""
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected missing evidence error")
	}
}

func TestValidateConfigRejectsUnknownEvidenceType(t *testing.T) {
	cfg := validConfig()
	cfg.EvidenceType = "ai_guess"
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected invalid evidence type error")
	}
}

func TestValidateConfigRejectsInvalidConfidence(t *testing.T) {
	cfg := validConfig()
	cfg.Confidence = 1.1
	if err := validateConfig(cfg); err == nil {
		t.Fatal("expected invalid confidence error")
	}
}

func TestProposalEligibilityAllowsNewAndExistingSuggestedAlias(t *testing.T) {
	preview := postgresstore.CanonicalAliasProposalPreview{}
	eligible, reason := proposalEligibility(preview)
	if !eligible || reason != "" {
		t.Fatalf("expected new alias to be eligible, got eligible=%v reason=%q", eligible, reason)
	}

	preview.ExistingStatus = "suggested"
	eligible, reason = proposalEligibility(preview)
	if !eligible || reason != "" {
		t.Fatalf("expected suggested alias to accept more evidence, got eligible=%v reason=%q", eligible, reason)
	}
}

func TestProposalEligibilityRejectsAuthoritativeOrTerminalCases(t *testing.T) {
	tests := []struct {
		name    string
		preview postgresstore.CanonicalAliasProposalPreview
		reason  string
	}{
		{
			name:    "already canonical",
			preview: postgresstore.CanonicalAliasProposalPreview{AlreadyCanonical: true},
			reason:  "already_canonical",
		},
		{
			name: "verified conflict",
			preview: postgresstore.CanonicalAliasProposalPreview{
				VerifiedConflict: &catalog.CanonicalIngredient{ID: "otro"},
			},
			reason: "verified_conflict",
		},
		{
			name:    "already verified",
			preview: postgresstore.CanonicalAliasProposalPreview{ExistingStatus: "verified"},
			reason:  "already_verified",
		},
		{
			name:    "already rejected",
			preview: postgresstore.CanonicalAliasProposalPreview{ExistingStatus: "rejected"},
			reason:  "already_rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eligible, reason := proposalEligibility(tt.preview)
			if eligible || reason != tt.reason {
				t.Fatalf("expected ineligible reason %q, got eligible=%v reason=%q", tt.reason, eligible, reason)
			}
		})
	}
}
