package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/curation"
)

func TestClientProposeUsesStructuredResponsesAndAttachesTrustedAgentMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["model"] != "test-model" {
			t.Fatalf("unexpected model: %#v", body["model"])
		}
		if store, ok := body["store"].(bool); !ok || store {
			t.Fatalf("expected store=false, got %#v", body["store"])
		}
		text, ok := body["text"].(map[string]any)
		if !ok {
			t.Fatalf("missing text config: %#v", body["text"])
		}
		format, ok := text["format"].(map[string]any)
		if !ok || format["type"] != "json_schema" || format["strict"] != true {
			t.Fatalf("unexpected structured output config: %#v", text["format"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_test_123",
			"status":"completed",
			"output":[{
				"type":"message",
				"content":[{
					"type":"output_text",
					"text":"{\"schemaVersion\":\"canonical-curation:v1\",\"policyVersion\":\"canonical-curation-policy:v1\",\"action\":\"propose_alias\",\"alias\":\"Arroz de grano redondo\",\"canonicalIngredientId\":\"arroz_redondo\",\"confidence\":0.97,\"reasons\":[\"misma variedad\"],\"conflicts\":[],\"evidence\":[{\"type\":\"supermarket_product\",\"supermarketProductId\":\"product-1\",\"sourceRef\":\"\",\"sourceText\":\"\"}]}"
				}]
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		APIKey:  "test-key",
		Model:   "test-model",
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	proposal, err := client.Propose(context.Background(), curation.ModelCase{
		Alias:                 "Arroz de grano redondo",
		CanonicalIngredientID: "arroz_redondo",
		CanonicalName:         "Arroz redondo",
		Products: []curation.ModelProductContext{{
			ID:                 "product-1",
			SupermarketID:      "dia",
			Name:               "Arroz redondo 1 kg",
			SourceCategoryName: "Arroz",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Action != curation.ActionProposeAlias || proposal.Confidence != 0.97 {
		t.Fatalf("unexpected proposal: %+v", proposal)
	}
	if proposal.Agent.Provider != "openai" || proposal.Agent.Model != "test-model" || proposal.Agent.RunID != "resp_test_123" {
		t.Fatalf("unexpected trusted agent metadata: %+v", proposal.Agent)
	}
}

func TestClientProposeFailsClosedOnRefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_refusal",
			"status":"completed",
			"output":[{
				"type":"message",
				"content":[{"type":"refusal","refusal":"cannot comply"}]
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{APIKey: "test-key", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Propose(context.Background(), curation.ModelCase{}); err == nil {
		t.Fatal("expected refusal to fail closed")
	}
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	if _, err := NewClient(Config{}); err == nil {
		t.Fatal("expected missing API key error")
	}
}
