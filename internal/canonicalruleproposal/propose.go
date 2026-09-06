package canonicalruleproposal

import (
	"math"
	"sort"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalanalysis"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type Options struct {
	MinSupport       int
	MinConcentration float64
	MaxPerFamily     int
}

type Proposal struct {
	Family             string   `json:"family"`
	Pattern            string   `json:"pattern"`
	SuggestedConceptID string   `json:"suggestedConceptId"`
	Decision           string   `json:"decision"`
	RequiredTerms      []string `json:"requiredTerms"`
	IdentityModifiers  []string `json:"identityModifiers,omitempty"`
	CategoryBoosts     []string `json:"categoryBoosts,omitempty"`
	CategoryVetoes     []string `json:"categoryVetoes,omitempty"`
	PreparedVetoes     []string `json:"preparedVetoes,omitempty"`
	SupportCount       int      `json:"supportCount"`
	Concentration      float64  `json:"concentration"`
	EvidenceScore      float64  `json:"evidenceScore"`
	Examples           []string `json:"examples,omitempty"`
}

type FamilyReport struct {
	Family         string     `json:"family"`
	Products       int        `json:"products"`
	CategoryVetoes []string   `json:"categoryVetoes,omitempty"`
	PreparedVetoes []string   `json:"preparedVetoes,omitempty"`
	Proposals      []Proposal `json:"proposals"`
}

type Report struct {
	ProductsTotal int            `json:"productsTotal"`
	ProductsUsed  int            `json:"productsUsed"`
	Families      []FamilyReport `json:"families"`
}

var preparedRoots = map[string]struct{}{
	"platos-preparados-y-pizzas": {},
}

func Propose(analysis canonicalanalysis.Report, options Options) Report {
	options = normalizeOptions(options)
	out := Report{ProductsTotal: analysis.ProductsTotal, ProductsUsed: analysis.ProductsUsed}

	categoryReports := make(map[string]canonicalanalysis.CategoryReport, len(analysis.Categories))
	for _, category := range analysis.Categories {
		categoryReports[category.Path] = category
	}

	for _, anchorContext := range analysis.AnchorContexts {
		family := catalog.NormalizeSearchText(anchorContext.Anchor)
		if family == "" {
			continue
		}

		categoryVetoes := preparedCategoryVetoes(anchorContext.Categories)
		preparedVetoes := preparedPatterns(family, analysis.Categories, options.MinSupport)
		familyReport := FamilyReport{
			Family: family,
			Products: anchorContext.Products,
			CategoryVetoes: categoryVetoes,
			PreparedVetoes: preparedVetoes,
		}

		for _, pattern := range analysis.GlobalPatterns {
			if pattern.N < 2 || pattern.Count < options.MinSupport || pattern.Concentration < options.MinConcentration {
				continue
			}
			terms := strings.Fields(catalog.NormalizeSearchText(pattern.Text))
			if !containsAllTerms(terms, strings.Fields(family)) {
				continue
			}

			modifiers := subtractTerms(terms, strings.Fields(family))
			if len(modifiers) == 0 {
				continue
			}

			proposal := Proposal{
				Family: family,
				Pattern: pattern.Text,
				SuggestedConceptID: strings.Join(terms, "_"),
				Decision: "review",
				RequiredTerms: append([]string(nil), terms...),
				IdentityModifiers: modifiers,
				CategoryVetoes: append([]string(nil), categoryVetoes...),
				PreparedVetoes: append([]string(nil), preparedVetoes...),
				SupportCount: pattern.Count,
				Concentration: round4(pattern.Concentration),
				EvidenceScore: evidenceScore(pattern),
				Examples: append([]string(nil), pattern.Examples...),
			}
			if pattern.DominantCategory != "" {
				if _, exists := categoryReports[pattern.DominantCategory]; exists {
					proposal.CategoryBoosts = []string{pattern.DominantCategory}
				}
			}
			familyReport.Proposals = append(familyReport.Proposals, proposal)
		}

		sort.Slice(familyReport.Proposals, func(i, j int) bool {
			if familyReport.Proposals[i].EvidenceScore != familyReport.Proposals[j].EvidenceScore {
				return familyReport.Proposals[i].EvidenceScore > familyReport.Proposals[j].EvidenceScore
			}
			if familyReport.Proposals[i].SupportCount != familyReport.Proposals[j].SupportCount {
				return familyReport.Proposals[i].SupportCount > familyReport.Proposals[j].SupportCount
			}
			return familyReport.Proposals[i].Pattern < familyReport.Proposals[j].Pattern
		})
		if len(familyReport.Proposals) > options.MaxPerFamily {
			familyReport.Proposals = familyReport.Proposals[:options.MaxPerFamily]
		}
		out.Families = append(out.Families, familyReport)
	}

	sort.Slice(out.Families, func(i, j int) bool { return out.Families[i].Family < out.Families[j].Family })
	return out
}

func normalizeOptions(options Options) Options {
	if options.MinSupport <= 0 {
		options.MinSupport = 3
	}
	if options.MinConcentration <= 0 || options.MinConcentration > 1 {
		options.MinConcentration = 0.70
	}
	if options.MaxPerFamily <= 0 {
		options.MaxPerFamily = 20
	}
	return options
}

func preparedCategoryVetoes(categories map[string]int) []string {
	var out []string
	for path, count := range categories {
		if count <= 0 || !isPreparedPath(path) {
			continue
		}
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func preparedPatterns(anchor string, categories []canonicalanalysis.CategoryReport, minSupport int) []string {
	seen := make(map[string]struct{})
	for _, category := range categories {
		if !isPreparedPath(category.Path) {
			continue
		}
		for _, pattern := range category.TopPatterns {
			if pattern.N < 2 || pattern.Count < minSupport {
				continue
			}
			terms := strings.Fields(catalog.NormalizeSearchText(pattern.Text))
			if !containsAllTerms(terms, strings.Fields(anchor)) {
				continue
			}
			seen[pattern.Text] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for pattern := range seen {
		out = append(out, pattern)
	}
	sort.Strings(out)
	return out
}

func isPreparedPath(path string) bool {
	path = strings.Trim(strings.TrimSpace(path), "/")
	root := path
	if i := strings.IndexByte(path, '/'); i >= 0 {
		root = path[:i]
	}
	_, ok := preparedRoots[root]
	return ok
}

func containsAllTerms(haystack, needle []string) bool {
	set := make(map[string]struct{}, len(haystack))
	for _, term := range haystack {
		set[term] = struct{}{}
	}
	for _, term := range needle {
		if _, ok := set[term]; !ok {
			return false
		}
	}
	return true
}

func subtractTerms(terms, family []string) []string {
	remaining := make(map[string]int)
	for _, term := range family {
		remaining[term]++
	}
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		if remaining[term] > 0 {
			remaining[term]--
			continue
		}
		out = append(out, term)
	}
	return out
}

func evidenceScore(pattern canonicalanalysis.Pattern) float64 {
	support := math.Min(1, float64(pattern.Count)/10)
	specificity := math.Min(1, float64(pattern.N)/3)
	return round4(0.50*pattern.Concentration + 0.30*support + 0.20*specificity)
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}
