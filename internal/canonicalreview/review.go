package canonicalreview

import (
	"sort"
	"strings"

	canonicaltext "github.com/WolcenOn/Supermarket-Prices-API/internal/canonical"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalgaps"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalsemantics"
)

type Priority string

const (
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

type Candidate struct {
	Family             string                  `json:"family"`
	Role               canonicalsemantics.Role `json:"role"`
	SuggestedConceptID string                  `json:"suggestedConceptId"`
	DisplayName        string                  `json:"displayName"`
	SemanticKey        string                  `json:"semanticKey"`
	Priority           Priority                `json:"priority"`
	SupportTotal       int                     `json:"supportTotal"`
	SupportMax         int                     `json:"supportMax"`
	EvidenceMax        float64                 `json:"evidenceMax"`
	Patterns           []string                `json:"patterns"`
	SourceConceptIDs   []string                `json:"sourceConceptIds"`
	Reasons            []string                `json:"reasons"`
}

type Summary struct {
	MissingItems       int `json:"missingItems"`
	GroupedCandidates int `json:"groupedCandidates"`
	HighPriority      int `json:"highPriority"`
	MediumPriority    int `json:"mediumPriority"`
	LowPriority       int `json:"lowPriority"`
}

type Report struct {
	ProductsTotal int         `json:"productsTotal"`
	ProductsUsed  int         `json:"productsUsed"`
	Summary       Summary     `json:"summary"`
	Candidates    []Candidate `json:"candidates"`
}

type bucket struct {
	family           string
	role             canonicalsemantics.Role
	conceptIDs       map[string]struct{}
	patterns         map[string]struct{}
	reasons          map[string]struct{}
	supportTotal     int
	supportMax       int
	evidenceMax      float64
}

func Build(input canonicalgaps.Report) Report {
	out := Report{ProductsTotal: input.ProductsTotal, ProductsUsed: input.ProductsUsed}
	groups := map[string]*bucket{}

	for _, item := range input.Items {
		if item.Status != canonicalgaps.StatusMissing {
			continue
		}
		out.Summary.MissingItems++
		key := groupKey(item)
		b := groups[key]
		if b == nil {
			b = &bucket{
				family: item.Family,
				role: item.Role,
				conceptIDs: map[string]struct{}{},
				patterns: map[string]struct{}{},
				reasons: map[string]struct{}{},
			}
			groups[key] = b
		}
		if id := strings.TrimSpace(item.SuggestedConceptID); id != "" {
			b.conceptIDs[id] = struct{}{}
		}
		if p := strings.TrimSpace(item.Pattern); p != "" {
			b.patterns[p] = struct{}{}
		}
		for _, reason := range item.Reason {
			if reason = strings.TrimSpace(reason); reason != "" {
				b.reasons[reason] = struct{}{}
			}
		}
		b.supportTotal += item.SupportCount
		if item.SupportCount > b.supportMax {
			b.supportMax = item.SupportCount
		}
		if item.EvidenceScore > b.evidenceMax {
			b.evidenceMax = item.EvidenceScore
		}
	}

	for key, b := range groups {
		ids := sortedSet(b.conceptIDs)
		patterns := sortedSet(b.patterns)
		reasons := sortedSet(b.reasons)
		conceptID := preferredConceptID(ids, patterns)
		candidate := Candidate{
			Family: b.family,
			Role: b.role,
			SuggestedConceptID: conceptID,
			DisplayName: displayName(conceptID, patterns),
			SemanticKey: key,
			Priority: priorityFor(b.supportTotal, b.supportMax, b.evidenceMax),
			SupportTotal: b.supportTotal,
			SupportMax: b.supportMax,
			EvidenceMax: round4(b.evidenceMax),
			Patterns: patterns,
			SourceConceptIDs: ids,
			Reasons: reasons,
		}
		out.Candidates = append(out.Candidates, candidate)
		switch candidate.Priority {
		case PriorityHigh:
			out.Summary.HighPriority++
		case PriorityMedium:
			out.Summary.MediumPriority++
		default:
			out.Summary.LowPriority++
		}
	}
	out.Summary.GroupedCandidates = len(out.Candidates)

	sort.Slice(out.Candidates, func(i, j int) bool {
		if priorityRank(out.Candidates[i].Priority) != priorityRank(out.Candidates[j].Priority) {
			return priorityRank(out.Candidates[i].Priority) < priorityRank(out.Candidates[j].Priority)
		}
		if out.Candidates[i].EvidenceMax != out.Candidates[j].EvidenceMax {
			return out.Candidates[i].EvidenceMax > out.Candidates[j].EvidenceMax
		}
		if out.Candidates[i].SupportTotal != out.Candidates[j].SupportTotal {
			return out.Candidates[i].SupportTotal > out.Candidates[j].SupportTotal
		}
		return out.Candidates[i].SuggestedConceptID < out.Candidates[j].SuggestedConceptID
	})
	return out
}

func groupKey(item canonicalgaps.Item) string {
	parts := semanticTerms(item.Pattern)
	if len(parts) == 0 {
		parts = semanticTerms(item.SuggestedConceptID)
	}
	return canonicaltext.NormalizeText(item.Family) + "|" + strings.Join(parts, " ")
}

func semanticTerms(value string) []string {
	value = strings.ReplaceAll(value, "_", " ")
	fields := strings.Fields(canonicaltext.NormalizeText(value))
	seen := map[string]struct{}{}
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, skip := connectorTerms[field]; skip {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	sort.Strings(out)
	return out
}

var connectorTerms = map[string]struct{}{
	"de": {}, "del": {}, "la": {}, "el": {}, "los": {}, "las": {}, "y": {}, "e": {},
}

func preferredConceptID(ids, patterns []string) string {
	if len(ids) > 0 {
		sort.Slice(ids, func(i, j int) bool {
			if len(ids[i]) != len(ids[j]) { return len(ids[i]) < len(ids[j]) }
			return ids[i] < ids[j]
		})
		return ids[0]
	}
	if len(patterns) == 0 {
		return ""
	}
	return strings.ReplaceAll(canonicaltext.NormalizeText(patterns[0]), " ", "_")
}

func displayName(conceptID string, patterns []string) string {
	if len(patterns) > 0 {
		return patterns[0]
	}
	return strings.ReplaceAll(conceptID, "_", " ")
}

func priorityFor(total, max int, evidence float64) Priority {
	if evidence >= 0.85 && (max >= 5 || total >= 8) {
		return PriorityHigh
	}
	if evidence >= 0.75 && (max >= 3 || total >= 4) {
		return PriorityMedium
	}
	return PriorityLow
}

func sortedSet(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func priorityRank(value Priority) int {
	switch value {
	case PriorityHigh:
		return 0
	case PriorityMedium:
		return 1
	default:
		return 2
	}
}

func round4(value float64) float64 {
	if value == 0 { return 0 }
	return float64(int(value*10000+0.5)) / 10000
}
