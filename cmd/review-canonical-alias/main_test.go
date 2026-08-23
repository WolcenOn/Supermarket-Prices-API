package main

import (
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

func TestValidateConfig(t *testing.T) {
	valid := config{
		AliasID:      1,
		Decision:     "verified",
		ReviewSource: "manual:cli",
		Note:         "Reviewed against source taxonomy and product examples",
	}
	if err := validateConfig(valid); err != nil {
		t.Fatalf("expected valid config, got %v", err)
	}

	cases := []struct {
		name string
		cfg  config
	}{
		{name: "missing alias id", cfg: config{Decision: "verified", ReviewSource: "manual:cli", Note: "note"}},
		{name: "invalid decision", cfg: config{AliasID: 1, Decision: "maybe", ReviewSource: "manual:cli", Note: "note"}},
		{name: "missing source", cfg: config{AliasID: 1, Decision: "verified", Note: "note"}},
		{name: "missing note", cfg: config{AliasID: 1, Decision: "verified", ReviewSource: "manual:cli"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateConfig(tc.cfg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestReviewEligibilityVerifiedRequiresSuggestedEvidenceAndNoConflict(t *testing.T) {
	base := postgresstore.CanonicalAliasReviewPreview{
		Alias: catalog.CanonicalIngredientAlias{ID: 1, Status: "suggested"},
		EvidenceCount: 1,
	}
	if ok, reason := reviewEligibility(base, "verified"); !ok || reason != "" {
		t.Fatalf("expected eligible verified review, got ok=%v reason=%q", ok, reason)
	}

	noEvidence := base
	noEvidence.EvidenceCount = 0
	if ok, reason := reviewEligibility(noEvidence, "verified"); ok || reason != "missing_evidence" {
		t.Fatalf("expected missing_evidence, got ok=%v reason=%q", ok, reason)
	}

	conflict := base
	conflict.VerifiedConflict = &catalog.CanonicalIngredient{ID: "other"}
	if ok, reason := reviewEligibility(conflict, "verified"); ok || reason != "verified_conflict" {
		t.Fatalf("expected verified_conflict, got ok=%v reason=%q", ok, reason)
	}

	alreadyVerified := base
	alreadyVerified.Alias.Status = "verified"
	if ok, reason := reviewEligibility(alreadyVerified, "verified"); ok || reason != "alias_not_suggested" {
		t.Fatalf("expected alias_not_suggested, got ok=%v reason=%q", ok, reason)
	}
}

func TestReviewEligibilityRejectedDoesNotRequireEvidence(t *testing.T) {
	preview := postgresstore.CanonicalAliasReviewPreview{
		Alias: catalog.CanonicalIngredientAlias{ID: 1, Status: "suggested"},
	}
	if ok, reason := reviewEligibility(preview, "rejected"); !ok || reason != "" {
		t.Fatalf("expected rejected review to be eligible, got ok=%v reason=%q", ok, reason)
	}
}

func TestMode(t *testing.T) {
	if got := mode(false); got != "preview" {
		t.Fatalf("expected preview, got %q", got)
	}
	if got := mode(true); got != "persist" {
		t.Fatalf("expected persist, got %q", got)
	}
}
