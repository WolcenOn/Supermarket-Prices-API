package canonicalcoverage

import (
	"context"
	"sort"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const (
	StatusCanonicalExact = "canonical_exact"
	StatusVerifiedAlias  = "verified_alias"
	StatusSuggestedAlias = "suggested_alias"
	StatusAmbiguous      = "ambiguous"
	StatusUnresolved     = "unresolved"
)

type Resolver interface {
	ResolveCanonicalIngredient(ctx context.Context, query string) (catalog.CanonicalResolution, error)
}

type Input struct {
	Name   string `json:"name"`
	Count  int    `json:"count,omitempty"`
	Source string `json:"source,omitempty"`
}

type Item struct {
	Name          string                                 `json:"name"`
	Normalized    string                                 `json:"normalized"`
	Count         int                                    `json:"count"`
	Sources       []string                               `json:"sources,omitempty"`
	Status        string                                 `json:"status"`
	CanonicalID   string                                 `json:"canonicalId,omitempty"`
	CanonicalName string                                 `json:"canonicalName,omitempty"`
	Confidence    float64                                `json:"confidence,omitempty"`
	Candidates    []catalog.CanonicalResolutionCandidate `json:"candidates,omitempty"`
}

type Summary struct {
	UniqueIngredients int     `json:"uniqueIngredients"`
	TotalOccurrences  int     `json:"totalOccurrences"`
	ResolvedUnique    int     `json:"resolvedUnique"`
	ResolvedOccurrences int   `json:"resolvedOccurrences"`
	CoverageUnique    float64 `json:"coverageUnique"`
	CoverageOccurrences float64 `json:"coverageOccurrences"`
	CanonicalExact    int     `json:"canonicalExact"`
	VerifiedAlias     int     `json:"verifiedAlias"`
	SuggestedAlias    int     `json:"suggestedAlias"`
	Ambiguous         int     `json:"ambiguous"`
	Unresolved        int     `json:"unresolved"`
}

type Report struct {
	Summary Summary `json:"summary"`
	Items   []Item  `json:"items"`
}

func Analyze(ctx context.Context, resolver Resolver, inputs []Input) (Report, error) {
	aggregated := aggregate(inputs)
	items := make([]Item, 0, len(aggregated))
	var summary Summary

	for _, input := range aggregated {
		resolution, err := resolver.ResolveCanonicalIngredient(ctx, input.Name)
		if err != nil {
			return Report{}, err
		}

		item := classify(input, resolution)
		items = append(items, item)
		summary.UniqueIngredients++
		summary.TotalOccurrences += item.Count
		switch item.Status {
		case StatusCanonicalExact:
			summary.CanonicalExact++
			summary.ResolvedUnique++
			summary.ResolvedOccurrences += item.Count
		case StatusVerifiedAlias:
			summary.VerifiedAlias++
			summary.ResolvedUnique++
			summary.ResolvedOccurrences += item.Count
		case StatusSuggestedAlias:
			summary.SuggestedAlias++
		case StatusAmbiguous:
			summary.Ambiguous++
		default:
			summary.Unresolved++
		}
	}

	if summary.UniqueIngredients > 0 {
		summary.CoverageUnique = float64(summary.ResolvedUnique) / float64(summary.UniqueIngredients)
	}
	if summary.TotalOccurrences > 0 {
		summary.CoverageOccurrences = float64(summary.ResolvedOccurrences) / float64(summary.TotalOccurrences)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return statusRank(items[i].Status) < statusRank(items[j].Status)
		}
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Name < items[j].Name
	})

	return Report{Summary: summary, Items: items}, nil
}

func classify(input Input, resolution catalog.CanonicalResolution) Item {
	item := Item{
		Name:       input.Name,
		Normalized: resolution.NormalizedQuery,
		Count:      input.Count,
		Sources:    splitSources(input.Source),
		Status:     StatusUnresolved,
		Candidates: resolution.Candidates,
	}

	if len(resolution.Candidates) == 0 {
		return item
	}

	if resolution.Status == "verified" {
		if len(resolution.Candidates) != 1 {
			item.Status = StatusAmbiguous
			return item
		}
		candidate := resolution.Candidates[0]
		item.CanonicalID = candidate.Ingredient.ID
		item.CanonicalName = candidate.Ingredient.Name
		item.Confidence = candidate.Confidence
		if candidate.MatchType == "canonical_name" {
			item.Status = StatusCanonicalExact
		} else {
			item.Status = StatusVerifiedAlias
		}
		return item
	}

	if resolution.Status == "suggested" {
		if len(resolution.Candidates) == 1 {
			candidate := resolution.Candidates[0]
			item.Status = StatusSuggestedAlias
			item.CanonicalID = candidate.Ingredient.ID
			item.CanonicalName = candidate.Ingredient.Name
			item.Confidence = candidate.Confidence
		} else {
			item.Status = StatusAmbiguous
		}
	}
	return item
}

func aggregate(inputs []Input) []Input {
	type bucket struct {
		name    string
		count   int
		sources map[string]struct{}
	}
	buckets := map[string]*bucket{}
	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			continue
		}
		count := input.Count
		if count <= 0 {
			count = 1
		}
		key := strings.ToLower(name)
		entry := buckets[key]
		if entry == nil {
			entry = &bucket{name: name, sources: map[string]struct{}{}}
			buckets[key] = entry
		}
		entry.count += count
		if source := strings.TrimSpace(input.Source); source != "" {
			entry.sources[source] = struct{}{}
		}
	}

	result := make([]Input, 0, len(buckets))
	for _, entry := range buckets {
		sources := make([]string, 0, len(entry.sources))
		for source := range entry.sources {
			sources = append(sources, source)
		}
		sort.Strings(sources)
		result = append(result, Input{Name: entry.name, Count: entry.count, Source: strings.Join(sources, "\n")})
	}
	sort.Slice(result, func(i, j int) bool { return strings.ToLower(result[i].Name) < strings.ToLower(result[j].Name) })
	return result
}

func splitSources(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, "\n")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func statusRank(status string) int {
	switch status {
	case StatusUnresolved:
		return 0
	case StatusAmbiguous:
		return 1
	case StatusSuggestedAlias:
		return 2
	case StatusVerifiedAlias:
		return 3
	case StatusCanonicalExact:
		return 4
	default:
		return 5
	}
}
