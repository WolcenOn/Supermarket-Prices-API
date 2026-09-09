package canonicalgaps

import (
	"context"
	"sort"
	"strings"

	canonicaltext "github.com/WolcenOn/Supermarket-Prices-API/internal/canonical"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalsemantics"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const (
	StatusCanonicalExact     = "canonical_exact"
	StatusVerifiedAlias      = "verified_alias"
	StatusSemanticEquivalent = "semantic_equivalent"
	StatusSuggestedAlias     = "suggested_alias"
	StatusAmbiguous          = "ambiguous"
	StatusMissing            = "missing"
)

type Store interface {
	Ingredients(ctx context.Context) ([]catalog.CanonicalIngredient, error)
	ResolveCanonicalIngredient(ctx context.Context, query string) (catalog.CanonicalResolution, error)
}

type Item struct {
	Family             string                                 `json:"family"`
	Role               canonicalsemantics.Role                `json:"role"`
	Pattern            string                                 `json:"pattern"`
	SuggestedConceptID string                                 `json:"suggestedConceptId"`
	SupportCount       int                                    `json:"supportCount"`
	EvidenceScore      float64                                `json:"evidenceScore"`
	Status             string                                 `json:"status"`
	CanonicalID        string                                 `json:"canonicalId,omitempty"`
	CanonicalName      string                                 `json:"canonicalName,omitempty"`
	MatchType          string                                 `json:"matchType,omitempty"`
	Candidates         []catalog.CanonicalResolutionCandidate `json:"candidates,omitempty"`
	Reason             []string                               `json:"reason,omitempty"`
}

type Summary struct {
	Candidates           int `json:"candidates"`
	CanonicalExact       int `json:"canonicalExact"`
	VerifiedAlias        int `json:"verifiedAlias"`
	SemanticEquivalent   int `json:"semanticEquivalent"`
	SuggestedAlias       int `json:"suggestedAlias"`
	Ambiguous            int `json:"ambiguous"`
	Missing              int `json:"missing"`
}

type Report struct {
	ProductsTotal int       `json:"productsTotal"`
	ProductsUsed  int       `json:"productsUsed"`
	Summary       Summary   `json:"summary"`
	Items         []Item    `json:"items"`
}

func Analyze(ctx context.Context, store Store, input canonicalsemantics.Report) (Report, error) {
	canonicals, err := store.Ingredients(ctx)
	if err != nil {
		return Report{}, err
	}
	fingerprints := buildFingerprintIndex(canonicals)
	out := Report{ProductsTotal: input.ProductsTotal, ProductsUsed: input.ProductsUsed}

	for _, family := range input.Families {
		for _, proposal := range family.Proposals {
			if proposal.Role != canonicalsemantics.RoleIngredientCandidate && proposal.Role != canonicalsemantics.RoleProcessedIngredient {
				continue
			}
			item, err := classify(ctx, store, fingerprints, family.Family, proposal)
			if err != nil {
				return Report{}, err
			}
			out.Items = append(out.Items, item)
			out.Summary.Candidates++
			switch item.Status {
			case StatusCanonicalExact:
				out.Summary.CanonicalExact++
			case StatusVerifiedAlias:
				out.Summary.VerifiedAlias++
			case StatusSemanticEquivalent:
				out.Summary.SemanticEquivalent++
			case StatusSuggestedAlias:
				out.Summary.SuggestedAlias++
			case StatusAmbiguous:
				out.Summary.Ambiguous++
			case StatusMissing:
				out.Summary.Missing++
			}
		}
	}

	sort.Slice(out.Items, func(i, j int) bool {
		if statusRank(out.Items[i].Status) != statusRank(out.Items[j].Status) {
			return statusRank(out.Items[i].Status) < statusRank(out.Items[j].Status)
		}
		if out.Items[i].EvidenceScore != out.Items[j].EvidenceScore {
			return out.Items[i].EvidenceScore > out.Items[j].EvidenceScore
		}
		if out.Items[i].SupportCount != out.Items[j].SupportCount {
			return out.Items[i].SupportCount > out.Items[j].SupportCount
		}
		return out.Items[i].SuggestedConceptID < out.Items[j].SuggestedConceptID
	})
	return out, nil
}

func classify(ctx context.Context, store Store, fingerprints map[string][]catalog.CanonicalIngredient, family string, proposal canonicalsemantics.AuditedProposal) (Item, error) {
	item := Item{
		Family: family, Role: proposal.Role, Pattern: proposal.Pattern,
		SuggestedConceptID: proposal.SuggestedConceptID,
		SupportCount: proposal.SupportCount, EvidenceScore: proposal.EvidenceScore,
		Status: StatusMissing,
	}

	queries := uniqueQueries(proposal.SuggestedConceptID, proposal.Pattern)
	var suggested []catalog.CanonicalResolutionCandidate
	for _, query := range queries {
		resolution, err := store.ResolveCanonicalIngredient(ctx, query)
		if err != nil {
			return Item{}, err
		}
		if resolution.Status == "verified" {
			if len(resolution.Candidates) == 1 {
				candidate := resolution.Candidates[0]
				item.CanonicalID = candidate.Ingredient.ID
				item.CanonicalName = candidate.Ingredient.Name
				item.Candidates = resolution.Candidates
				item.MatchType = candidate.MatchType
				if candidate.MatchType == "canonical_name" {
					item.Status = StatusCanonicalExact
				} else {
					item.Status = StatusVerifiedAlias
				}
				return item, nil
			}
			if len(resolution.Candidates) > 1 {
				item.Status = StatusAmbiguous
				item.Candidates = resolution.Candidates
				item.Reason = []string{"multiple_verified_candidates"}
				return item, nil
			}
		}
		if resolution.Status == "suggested" {
			suggested = appendUniqueCandidates(suggested, resolution.Candidates...)
		}
	}

	fingerprint := semanticFingerprint(proposal.Pattern)
	if fingerprint == "" {
		fingerprint = semanticFingerprint(proposal.SuggestedConceptID)
	}
	matches := fingerprints[fingerprint]
	if len(matches) == 1 {
		item.Status = StatusSemanticEquivalent
		item.CanonicalID = matches[0].ID
		item.CanonicalName = matches[0].Name
		item.MatchType = "semantic_fingerprint"
		item.Reason = []string{"same_semantic_terms_ignoring_connectors_and_order"}
		return item, nil
	}
	if len(matches) > 1 {
		item.Status = StatusAmbiguous
		item.MatchType = "semantic_fingerprint"
		item.Reason = []string{"multiple_canonicals_share_semantic_fingerprint"}
		for _, ingredient := range matches {
			item.Candidates = append(item.Candidates, catalog.CanonicalResolutionCandidate{Ingredient: ingredient, MatchType: "semantic_fingerprint", Confidence: 1})
		}
		return item, nil
	}

	if len(suggested) == 1 {
		item.Status = StatusSuggestedAlias
		item.CanonicalID = suggested[0].Ingredient.ID
		item.CanonicalName = suggested[0].Ingredient.Name
		item.MatchType = suggested[0].MatchType
		item.Candidates = suggested
		item.Reason = []string{"single_suggested_alias"}
		return item, nil
	}
	if len(suggested) > 1 {
		item.Status = StatusAmbiguous
		item.Candidates = suggested
		item.Reason = []string{"multiple_suggested_aliases"}
		return item, nil
	}

	item.Reason = []string{"no_canonical_name_id_alias_or_semantic_equivalent"}
	return item, nil
}

func buildFingerprintIndex(items []catalog.CanonicalIngredient) map[string][]catalog.CanonicalIngredient {
	out := make(map[string][]catalog.CanonicalIngredient)
	for _, item := range items {
		seen := make(map[string]struct{})
		for _, value := range []string{item.ID, item.Name} {
			fingerprint := semanticFingerprint(value)
			if fingerprint == "" {
				continue
			}
			if _, ok := seen[fingerprint]; ok {
				continue
			}
			seen[fingerprint] = struct{}{}
			out[fingerprint] = append(out[fingerprint], item)
		}
	}
	return out
}

var connectorTerms = map[string]struct{}{
	"de": {}, "del": {}, "la": {}, "el": {}, "los": {}, "las": {}, "y": {}, "e": {},
}

func semanticFingerprint(value string) string {
	normalized := canonicaltext.NormalizeText(strings.ReplaceAll(value, "_", " "))
	terms := make([]string, 0)
	seen := make(map[string]struct{})
	for _, term := range strings.Fields(normalized) {
		if _, skip := connectorTerms[term]; skip {
			continue
		}
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}
	sort.Strings(terms)
	return strings.Join(terms, " ")
}

func uniqueQueries(values ...string) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := canonicaltext.NormalizeText(strings.ReplaceAll(value, "_", " "))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func appendUniqueCandidates(dst []catalog.CanonicalResolutionCandidate, values ...catalog.CanonicalResolutionCandidate) []catalog.CanonicalResolutionCandidate {
	seen := make(map[string]struct{}, len(dst))
	for _, candidate := range dst {
		seen[candidate.Ingredient.ID] = struct{}{}
	}
	for _, candidate := range values {
		if _, ok := seen[candidate.Ingredient.ID]; ok {
			continue
		}
		seen[candidate.Ingredient.ID] = struct{}{}
		dst = append(dst, candidate)
	}
	return dst
}

func statusRank(status string) int {
	switch status {
	case StatusMissing:
		return 0
	case StatusAmbiguous:
		return 1
	case StatusSuggestedAlias:
		return 2
	case StatusSemanticEquivalent:
		return 3
	case StatusVerifiedAlias:
		return 4
	case StatusCanonicalExact:
		return 5
	default:
		return 6
	}
}
