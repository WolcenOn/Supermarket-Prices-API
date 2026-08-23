package api

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type canonicalResolverTestStore struct {
    *catalog.MemoryStore
    resolution catalog.CanonicalResolution
}

func (s *canonicalResolverTestStore) ResolveCanonicalIngredient(_ context.Context, query string) (catalog.CanonicalResolution, error) {
    result := s.resolution
    result.Query = query
    return result, nil
}

func TestCanonicalResolverRequiresQuery(t *testing.T) {
    store := &canonicalResolverTestStore{MemoryStore: catalog.NewMemoryStore(nil)}
    h := NewHandler(store)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/ingredients/resolve", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusBadRequest {
        t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
    }
}

func TestCanonicalResolverExposesVerifiedAlias(t *testing.T) {
    alias := &catalog.CanonicalIngredientAlias{
        ID:                    7,
        CanonicalIngredientID: "arroz_redondo",
        Alias:                 "arroz de grano redondo",
        NormalizedAlias:       "arroz de grano redondo",
        Status:                "verified",
        Confidence:            0.99,
        DecisionSource:        "manual-review",
    }
    store := &canonicalResolverTestStore{
        MemoryStore: catalog.NewMemoryStore(nil),
        resolution: catalog.CanonicalResolution{
            NormalizedQuery: "arroz de grano redondo",
            Status:          "verified",
            Candidates: []catalog.CanonicalResolutionCandidate{{
                Ingredient: catalog.CanonicalIngredient{
                    ID:          "arroz_redondo",
                    Name:        "Arroz redondo",
                    Category:    "arroz",
                    Subtype:     "redondo",
                    DefaultUnit: "g",
                },
                Alias:      alias,
                MatchType:  "alias",
                Confidence: 0.99,
            }},
        },
    }

    h := NewHandler(store)
    req := httptest.NewRequest(http.MethodGet, "/api/v1/ingredients/resolve?q=arroz%20de%20grano%20redondo", nil)
    rec := httptest.NewRecorder()
    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
    }

    var body catalog.CanonicalResolution
    if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
        t.Fatal(err)
    }
    if body.Status != "verified" || len(body.Candidates) != 1 {
        t.Fatalf("unexpected resolution: %#v", body)
    }
    candidate := body.Candidates[0]
    if candidate.Ingredient.ID != "arroz_redondo" || candidate.MatchType != "alias" || candidate.Alias == nil {
        t.Fatalf("unexpected candidate: %#v", candidate)
    }
    if candidate.Alias.Status != "verified" || candidate.Confidence != 0.99 {
        t.Fatalf("verification metadata missing: %#v", candidate)
    }
}

func TestCanonicalResolverUnavailableWithoutResolverStore(t *testing.T) {
    h := NewHandler(catalog.NewMemoryStore(nil))
    req := httptest.NewRequest(http.MethodGet, "/api/v1/ingredients/resolve?q=arroz", nil)
    rec := httptest.NewRecorder()

    h.Routes().ServeHTTP(rec, req)

    if rec.Code != http.StatusServiceUnavailable {
        t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
    }
}
