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

type config struct {
	AliasID      int64
	Decision     string
	ReviewSource string
	Note         string
	Persist      bool
}

type output struct {
	Mode         string                                   `json:"mode"`
	Decision     string                                   `json:"decision"`
	ReviewSource string                                   `json:"reviewSource"`
	Note         string                                   `json:"note"`
	Preview      postgresstore.CanonicalAliasReviewPreview `json:"preview"`
	Eligible     bool                                     `json:"eligible"`
	Reason       string                                   `json:"reason,omitempty"`
	Saved        bool                                     `json:"saved"`
	Stored       *postgresstore.SavedCanonicalAliasReview `json:"stored,omitempty"`
}

func main() {
	aliasID := flag.Int64("alias-id", 0, "existing canonical alias id to review")
	decision := flag.String("decision", "", "review decision: verified or rejected")
	reviewSource := flag.String("review-source", "manual:cli", "review source, for example manual:cli or ai:curator-v1")
	note := flag.String("note", "", "required review note explaining the decision")
	persist := flag.Bool("persist", false, "persist the review decision; preview-only by default")
	timeout := flag.Duration("timeout", 30*time.Second, "maximum execution time")
	flag.Parse()

	cfg := config{
		AliasID:      *aliasID,
		Decision:     strings.TrimSpace(strings.ToLower(*decision)),
		ReviewSource: strings.TrimSpace(*reviewSource),
		Note:         strings.TrimSpace(*note),
		Persist:      *persist,
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

	preview, err := postgresstore.PreviewCanonicalAliasReview(ctx, db, cfg.AliasID)
	if err != nil {
		log.Fatal(err)
	}
	eligible, reason := reviewEligibility(preview, cfg.Decision)
	result := output{
		Mode:         mode(cfg.Persist),
		Decision:     cfg.Decision,
		ReviewSource: cfg.ReviewSource,
		Note:         cfg.Note,
		Preview:      preview,
		Eligible:     eligible,
		Reason:       reason,
	}

	if cfg.Persist {
		if !eligible {
			log.Fatalf("review is not eligible for persistence: %s", reason)
		}
		saved, err := postgresstore.SaveCanonicalAliasReview(ctx, db, postgresstore.CanonicalAliasReviewInput{
			AliasID:        cfg.AliasID,
			Decision:       cfg.Decision,
			DecisionSource: cfg.ReviewSource,
			Note:           cfg.Note,
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
	if cfg.AliasID <= 0 {
		return fmt.Errorf("--alias-id must be a positive integer")
	}
	if cfg.Decision != "verified" && cfg.Decision != "rejected" {
		return fmt.Errorf("--decision must be verified or rejected")
	}
	if cfg.ReviewSource == "" {
		return fmt.Errorf("--review-source is required")
	}
	if cfg.Note == "" {
		return fmt.Errorf("--note is required")
	}
	return nil
}

func reviewEligibility(preview postgresstore.CanonicalAliasReviewPreview, decision string) (bool, string) {
	if preview.Alias.Status != "suggested" {
		return false, "alias_not_suggested"
	}
	if decision == "verified" {
		if preview.EvidenceCount == 0 {
			return false, "missing_evidence"
		}
		if preview.VerifiedConflict != nil {
			return false, "verified_conflict"
		}
	}
	return true, ""
}

func mode(persist bool) string {
	if persist {
		return "persist"
	}
	return "preview"
}
