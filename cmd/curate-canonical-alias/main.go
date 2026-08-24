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

	"github.com/WolcenOn/Supermarket-Prices-API/internal/curation"
	openaiagent "github.com/WolcenOn/Supermarket-Prices-API/internal/curation/openai"
	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

type stringListFlag []string

func (f *stringListFlag) String() string {
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("value cannot be empty")
	}
	*f = append(*f, value)
	return nil
}

type output struct {
	Mode     string            `json:"mode"`
	Case     curation.ModelCase `json:"case"`
	Proposal curation.Proposal  `json:"proposal"`
	Verdict  curation.Verdict   `json:"verdict"`
}

func main() {
	canonicalID := flag.String("canonical-id", "", "existing canonical ingredient id")
	alias := flag.String("alias", "", "candidate equivalent alias to evaluate")
	model := flag.String("model", envOrDefault("OPENAI_CURATION_MODEL", openaiagent.DefaultModel), "OpenAI model used for semantic curation")
	baseURL := flag.String("openai-base-url", envOrDefault("OPENAI_BASE_URL", openaiagent.DefaultBaseURL), "OpenAI API base URL")
	timeout := flag.Duration("timeout", 90*time.Second, "maximum execution time")
	var productIDs stringListFlag
	flag.Var(&productIDs, "product-id", "existing supermarket product UUID used as evidence; repeat for multiple products")
	flag.Parse()

	canonicalIngredientID := strings.TrimSpace(*canonicalID)
	candidateAlias := strings.TrimSpace(*alias)
	if canonicalIngredientID == "" {
		log.Fatal("--canonical-id is required")
	}
	if candidateAlias == "" {
		log.Fatal("--alias is required")
	}
	productIDs = deduplicate(productIDs)
	if len(productIDs) == 0 {
		log.Fatal("at least one --product-id is required")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
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

	store := postgresstore.NewCatalogStore(db)
	modelCase, err := loadModelCase(ctx, store, canonicalIngredientID, candidateAlias, productIDs)
	if err != nil {
		log.Fatal(err)
	}

	client, err := openaiagent.NewClient(openaiagent.Config{
		APIKey:  apiKey,
		Model:   strings.TrimSpace(*model),
		BaseURL: strings.TrimSpace(*baseURL),
	})
	if err != nil {
		log.Fatal(err)
	}
	proposal, err := client.Propose(ctx, modelCase)
	if err != nil {
		log.Fatal(err)
	}
	verdict, err := curation.VerifyModelProposal(ctx, modelCase, proposal, store)
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output{
		Mode:     "read_only",
		Case:     modelCase,
		Proposal: proposal,
		Verdict:  verdict,
	}); err != nil {
		log.Fatal(err)
	}
}

func loadModelCase(ctx context.Context, store *postgresstore.CatalogStore, canonicalIngredientID, alias string, productIDs []string) (curation.ModelCase, error) {
	aliasContext, err := store.CurationAliasContext(ctx, canonicalIngredientID, alias)
	if err != nil {
		return curation.ModelCase{}, err
	}
	if aliasContext.AlreadyCanonical {
		return curation.ModelCase{}, fmt.Errorf("candidate alias already equals canonical ingredient %s", aliasContext.CanonicalIngredientID)
	}
	if aliasContext.ExistingStatus == "verified" {
		return curation.ModelCase{}, fmt.Errorf("candidate alias is already verified for %s", aliasContext.CanonicalIngredientID)
	}
	if aliasContext.ExistingStatus == "rejected" {
		return curation.ModelCase{}, fmt.Errorf("candidate alias was previously rejected for %s", aliasContext.CanonicalIngredientID)
	}
	if aliasContext.VerifiedConflictID != "" {
		return curation.ModelCase{}, fmt.Errorf("candidate alias conflicts with verified canonical ingredient %s", aliasContext.VerifiedConflictID)
	}

	products := make([]curation.ProductEvidence, 0, len(productIDs))
	for _, productID := range productIDs {
		product, found, err := store.CurationProductEvidence(ctx, productID)
		if err != nil {
			return curation.ModelCase{}, err
		}
		if !found {
			return curation.ModelCase{}, fmt.Errorf("supermarket product evidence %q not found", productID)
		}
		products = append(products, product)
	}

	return curation.ModelCase{
		Alias:                 strings.TrimSpace(alias),
		CanonicalIngredientID: aliasContext.CanonicalIngredientID,
		CanonicalName:         aliasContext.CanonicalName,
		Products:              products,
	}, nil
}

func deduplicate(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
