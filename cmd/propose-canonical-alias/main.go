package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

var allowedEvidenceTypes = map[string]struct{}{
	"supermarket_product": {},
	"source_taxonomy":     {},
	"pack":                {},
	"manual":              {},
	"rule":                {},
	"other":               {},
}

type config struct {
	CanonicalID   string
	Alias         string
	Confidence    float64
	DecisionSource string
	EvidenceType  string
	SourceRef     string
	SourceText    string
	Persist       bool
}

type output struct {
	Mode           string                                      `json:"mode"`
	ProposedStatus string                                      `json:"proposedStatus"`
	Confidence     float64                                     `json:"confidence"`
	DecisionSource string                                      `json:"decisionSource"`
	Evidence       evidenceOutput                              `json:"evidence"`
	Preview        postgresstore.CanonicalAliasProposalPreview `json:"preview"`
	Eligible       bool                                        `json:"eligible"`
	Reason         string                                      `json:"reason,omitempty"`
	Saved          bool                                        `json:"saved"`
	Stored         *postgresstore.SavedCanonicalAliasProposal  `json:"stored,omitempty"`
}

type evidenceOutput struct {
	Type       string `json:"type"`
	SourceRef  string `json:"sourceRef,omitempty"`
	SourceText string `json:"sourceText,omitempty"`
}

func main() {
	canonicalID := flag.String("canonical-id", "", "existing canonical ingredient id")
	alias := flag.String("alias", "", "human-readable alias to propose")
	confidence := flag.Float64("confidence", 0.90, "proposal confidence between 0 and 1")
	decisionSource := flag.String("decision-source", "manual:cli", "proposal source, for example manual:cli or ai:curator-v1")
	evidenceType := flag.String("evidence-type", "", "required evidence type: supermarket_product, source_taxonomy, pack, manual, rule, other")
	sourceRef := flag.String("source-ref", "", "stable evidence reference, for example a pack id or source URL")
	sourceText := flag.String("source-text", "", "human-readable evidence text")
	persist := flag.Bool("persist", false, "persist the proposal; preview-only by default")
	timeout := flag.Duration("timeout", 30*time.Second, "maximum execution time")
	flag.Parse()

	cfg := config{
		CanonicalID:    strings.TrimSpace(*canonicalID),
		Alias:          strings.TrimSpace(*alias),
		Confidence:     *confidence,
		DecisionSource: strings.TrimSpace(*decisionSource),
		EvidenceType:   strings.TrimSpace(*evidenceType),
		SourceRef:      strings.TrimSpace(*sourceRef),
		SourceText:     strings.TrimSpace(*sourceText),
		Persist:        *persist,
	}
	if err := validateConfig(cfg); err != nil {
		log.Fatal(err)
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	preview, err := postgresstore.PreviewCanonicalAliasProposal(ctx, db, cfg.CanonicalID, cfg.Alias)
	if err != nil {
		log.Fatal(err)
	}
	eligible, reason := proposalEligibility(preview)
	result := output{
		Mode:           mode(cfg.Persist),
		ProposedStatus: "suggested",
		Confidence:     cfg.Confidence,
		DecisionSource: cfg.DecisionSource,
		Evidence: evidenceOutput{
			Type:       cfg.EvidenceType,
			SourceRef:  cfg.SourceRef,
			SourceText: cfg.SourceText,
		},
		Preview:  preview,
		Eligible: eligible,
		Reason:   reason,
	}

	if cfg.Persist {
		if !eligible {
			log.Fatalf("proposal is not eligible for persistence: %s", reason)
		}
		saved, err := postgresstore.SaveCanonicalAliasProposal(ctx, db, postgresstore.CanonicalAliasProposalInput{
			CanonicalIngredientID: cfg.CanonicalID,
			Alias:                 cfg.Alias,
			Confidence:            cfg.Confidence,
			DecisionSource:        cfg.DecisionSource,
			EvidenceType:          cfg.EvidenceType,
			EvidenceSourceRef:     cfg.SourceRef,
			EvidenceSourceText:    cfg.SourceText,
		})
		if err != nil {
			log.Fatal(err)
		}
		result.Saved = true
		result.Stored = &saved
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		log.Fatal(err)
	}
}

func validateConfig(cfg config) error {
	if cfg.CanonicalID == "" {
		return fmt.Errorf("--canonical-id is required")
	}
	if cfg.Alias == "" {
		return fmt.Errorf("--alias is required")
	}
	if cfg.Confidence <= 0 || cfg.Confidence > 1 {
		return fmt.Errorf("--confidence must be greater than 0 and at most 1")
	}
	if cfg.DecisionSource == "" {
		return fmt.Errorf("--decision-source is required")
	}
	if _, ok := allowedEvidenceTypes[cfg.EvidenceType]; !ok {
		return fmt.Errorf("--evidence-type must be one of supermarket_product, source_taxonomy, pack, manual, rule, other")
	}
	if cfg.SourceRef == "" && cfg.SourceText == "" {
		return fmt.Errorf("at least one of --source-ref or --source-text is required")
	}
	return nil
}

func proposalEligibility(preview postgresstore.CanonicalAliasProposalPreview) (bool, string) {
	if preview.AlreadyCanonical {
		return false, "already_canonical"
	}
	if preview.VerifiedConflict != nil {
		return false, "verified_conflict"
	}
	switch preview.ExistingStatus {
	case "verified":
		return false, "already_verified"
	case "rejected":
		return false, "already_rejected"
	default:
		return true, ""
	}
}

func mode(persist bool) string {
	if persist {
		return "persist"
	}
	return "preview"
}
