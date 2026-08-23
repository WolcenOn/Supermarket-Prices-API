package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/curation"
	postgresstore "github.com/WolcenOn/Supermarket-Prices-API/internal/storage/postgres"
)

type output struct {
	Proposal curation.Proposal `json:"proposal"`
	Verdict  curation.Verdict  `json:"verdict"`
}

func main() {
	proposalJSON := flag.String("proposal-json", "", "canonical curation proposal JSON; when omitted, JSON is read from stdin")
	timeout := flag.Duration("timeout", 30*time.Second, "maximum execution time")
	flag.Parse()

	proposal, err := readProposal(*proposalJSON, os.Stdin)
	if err != nil {
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

	store := postgresstore.NewCatalogStore(db)
	verdict, err := curation.Verify(ctx, proposal, store)
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output{Proposal: proposal, Verdict: verdict}); err != nil {
		log.Fatal(err)
	}
}

func readProposal(raw string, reader io.Reader) (curation.Proposal, error) {
	var data []byte
	var err error
	if strings.TrimSpace(raw) != "" {
		data = []byte(raw)
	} else {
		data, err = io.ReadAll(reader)
		if err != nil {
			return curation.Proposal{}, fmt.Errorf("read proposal stdin: %w", err)
		}
	}
	if strings.TrimSpace(string(data)) == "" {
		return curation.Proposal{}, fmt.Errorf("proposal JSON is required via --proposal-json or stdin")
	}

	var proposal curation.Proposal
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proposal); err != nil {
		return curation.Proposal{}, fmt.Errorf("decode proposal JSON: %w", err)
	}
	return proposal, nil
}
