package canonicalanalysis

import (
	"sort"
	"strconv"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type Product struct {
	ExternalID string `json:"externalId"`
	Name       string `json:"name"`
	Category   struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"category"`
}

type Options struct {
	FoodOnly       bool
	MinSupport     int
	TopGlobal      int
	TopPerCategory int
	MaxExamples    int
	Anchors        []string
}

type Pattern struct {
	Text             string   `json:"text"`
	N                int      `json:"n"`
	Count            int      `json:"count"`
	DominantCategory string   `json:"dominantCategory,omitempty"`
	DominantCount    int      `json:"dominantCount,omitempty"`
	Concentration    float64  `json:"concentration"`
	Examples         []string `json:"examples,omitempty"`
}

type CategoryReport struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name,omitempty"`
	Path        string    `json:"path"`
	Root        string    `json:"root"`
	Products    int       `json:"products"`
	TopPatterns []Pattern `json:"topPatterns"`
}

type AnchorContext struct {
	Anchor     string            `json:"anchor"`
	Products   int               `json:"products"`
	Categories map[string]int    `json:"categories"`
	Contexts   []ContextPattern  `json:"contexts"`
}

type ContextPattern struct {
	Text     string   `json:"text"`
	Count    int      `json:"count"`
	Examples []string `json:"examples,omitempty"`
}

type Report struct {
	ProductsTotal int             `json:"productsTotal"`
	ProductsUsed  int             `json:"productsUsed"`
	Categories    []CategoryReport `json:"categories"`
	GlobalPatterns []Pattern       `json:"globalPatterns"`
	AnchorContexts []AnchorContext `json:"anchorContexts,omitempty"`
}

type categoryMeta struct {
	ID   string
	Name string
	Path string
	Root string
}

type patternStat struct {
	N          int
	Count      int
	Categories map[string]int
	Examples   []string
}

type contextStat struct {
	Count    int
	Examples []string
}

var defaultFoodRoots = map[string]struct{}{
	"aceites-salsas-y-especias": {},
	"aperitivos-y-frutos-secos": {},
	"arroz-pastas-y-legumbres": {},
	"bolleria-reposteria-y-azucar": {},
	"cafe-cacao-e-infusiones": {},
	"carnes": {},
	"charcuteria": {},
	"charcuteria-y-quesos": {},
	"chocolates-y-golosinas": {},
	"congelados": {},
	"congelados-y-helados": {},
	"conservas-caldos-y-cremas": {},
	"frutas": {},
	"galletas-cereales-y-mermeladas": {},
	"huevos-leche-y-mantequilla": {},
	"infantil": {},
	"panaderia": {},
	"pescados-y-mariscos": {},
	"platos-preparados-y-pizzas": {},
	"quesos": {},
	"verduras": {},
	"yogures-y-postres": {},
	"zumos-y-smoothies": {},
}

var stopTokens = map[string]struct{}{
	"de": {}, "del": {}, "la": {}, "el": {}, "los": {}, "las": {}, "y": {}, "e": {}, "con": {}, "en": {}, "al": {}, "a": {}, "por": {}, "para": {},
	"dia": {}, "seleccion": {}, "nuestra": {}, "alacena": {}, "vegecampo": {},
	"aprox": {}, "bandeja": {}, "formato": {}, "pack": {}, "unidad": {}, "unidades": {}, "ud": {}, "uds": {},
	"g": {}, "gr": {}, "kg": {}, "ml": {}, "cl": {}, "l": {}, "litro": {}, "litros": {},
	"x": {},
}

func Analyze(products []Product, options Options) Report {
	options = normalizeOptions(options)
	report := Report{ProductsTotal: len(products)}

	global := make(map[string]*patternStat)
	byCategory := make(map[string]map[string]*patternStat)
	categoryCounts := make(map[string]int)
	categoryInfo := make(map[string]categoryMeta)

	anchorTerms := make([]string, 0, len(options.Anchors))
	for _, anchor := range options.Anchors {
		anchor = catalog.NormalizeSearchText(anchor)
		if anchor != "" {
			anchorTerms = append(anchorTerms, anchor)
		}
	}
	anchorProducts := make(map[string]int)
	anchorCategories := make(map[string]map[string]int)
	anchorContexts := make(map[string]map[string]*contextStat)

	for _, product := range products {
		path := strings.TrimSpace(product.Category.Path)
		root := categoryRoot(path)
		if options.FoodOnly && !isFoodRoot(root) {
			continue
		}
		report.ProductsUsed++
		categoryCounts[path]++
		categoryInfo[path] = categoryMeta{ID: product.Category.ID, Name: product.Category.Name, Path: path, Root: root}

		tokens := semanticTokens(product.Name)
		seenGlobal := make(map[string]struct{})
		seenCategory := make(map[string]struct{})
		for n := 1; n <= 3; n++ {
			for _, phrase := range ngrams(tokens, n) {
				key := patternKey(n, phrase)
				if _, ok := seenGlobal[key]; !ok {
					seenGlobal[key] = struct{}{}
					stat := ensurePattern(global, key, n)
					stat.Count++
					stat.Categories[path]++
					appendExample(&stat.Examples, product.Name, options.MaxExamples)
				}
				if _, ok := seenCategory[key]; !ok {
					seenCategory[key] = struct{}{}
					if byCategory[path] == nil {
						byCategory[path] = make(map[string]*patternStat)
					}
					stat := ensurePattern(byCategory[path], key, n)
					stat.Count++
					appendExample(&stat.Examples, product.Name, options.MaxExamples)
				}
			}
		}

		normalizedNameTokens := strings.Fields(catalog.NormalizeSearchText(product.Name))
		for _, anchor := range anchorTerms {
			positions := tokenPositions(normalizedNameTokens, anchor)
			if len(positions) == 0 {
				continue
			}
			anchorProducts[anchor]++
			if anchorCategories[anchor] == nil {
				anchorCategories[anchor] = make(map[string]int)
			}
			anchorCategories[anchor][path]++
			if anchorContexts[anchor] == nil {
				anchorContexts[anchor] = make(map[string]*contextStat)
			}
			seenContext := make(map[string]struct{})
			for _, pos := range positions {
				start := pos - 2
				if start < 0 {
					start = 0
				}
				end := pos + 3
				if end > len(normalizedNameTokens) {
					end = len(normalizedNameTokens)
				}
				context := strings.Join(normalizedNameTokens[start:end], " ")
				if _, ok := seenContext[context]; ok {
					continue
				}
				seenContext[context] = struct{}{}
				stat := anchorContexts[anchor][context]
				if stat == nil {
					stat = &contextStat{}
					anchorContexts[anchor][context] = stat
				}
				stat.Count++
				appendExample(&stat.Examples, product.Name, options.MaxExamples)
			}
		}
	}

	report.GlobalPatterns = collectPatterns(global, options.MinSupport, options.TopGlobal, true)

	paths := make([]string, 0, len(categoryCounts))
	for path := range categoryCounts {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		if categoryCounts[paths[i]] != categoryCounts[paths[j]] {
			return categoryCounts[paths[i]] > categoryCounts[paths[j]]
		}
		return paths[i] < paths[j]
	})
	for _, path := range paths {
		meta := categoryInfo[path]
		report.Categories = append(report.Categories, CategoryReport{
			ID: meta.ID, Name: meta.Name, Path: meta.Path, Root: meta.Root,
			Products: categoryCounts[path],
			TopPatterns: collectPatterns(byCategory[path], options.MinSupport, options.TopPerCategory, false),
		})
	}

	for _, anchor := range anchorTerms {
		contextStats := anchorContexts[anchor]
		contexts := make([]ContextPattern, 0, len(contextStats))
		for text, stat := range contextStats {
			contexts = append(contexts, ContextPattern{Text: text, Count: stat.Count, Examples: stat.Examples})
		}
		sort.Slice(contexts, func(i, j int) bool {
			if contexts[i].Count != contexts[j].Count {
				return contexts[i].Count > contexts[j].Count
			}
			return contexts[i].Text < contexts[j].Text
		})
		if len(contexts) > options.TopPerCategory {
			contexts = contexts[:options.TopPerCategory]
		}
		report.AnchorContexts = append(report.AnchorContexts, AnchorContext{
			Anchor: anchor, Products: anchorProducts[anchor], Categories: anchorCategories[anchor], Contexts: contexts,
		})
	}

	return report
}

func normalizeOptions(options Options) Options {
	if options.MinSupport <= 0 {
		options.MinSupport = 2
	}
	if options.TopGlobal <= 0 {
		options.TopGlobal = 300
	}
	if options.TopPerCategory <= 0 {
		options.TopPerCategory = 15
	}
	if options.MaxExamples <= 0 {
		options.MaxExamples = 5
	}
	return options
}

func isFoodRoot(root string) bool {
	_, ok := defaultFoodRoots[root]
	return ok
}

func categoryRoot(path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return ""
	}
	if idx := strings.IndexByte(path, '/'); idx >= 0 {
		return path[:idx]
	}
	return path
}

func semanticTokens(value string) []string {
	normalized := catalog.NormalizeSearchText(value)
	fields := strings.Fields(normalized)
	out := make([]string, 0, len(fields))
	for _, token := range fields {
		if _, stop := stopTokens[token]; stop {
			continue
		}
		if _, err := strconv.ParseFloat(token, 64); err == nil {
			continue
		}
		if len(token) < 2 {
			continue
		}
		out = append(out, token)
	}
	return out
}

func ngrams(tokens []string, n int) []string {
	if n <= 0 || len(tokens) < n {
		return nil
	}
	out := make([]string, 0, len(tokens)-n+1)
	for i := 0; i+n <= len(tokens); i++ {
		out = append(out, strings.Join(tokens[i:i+n], " "))
	}
	return out
}

func patternKey(n int, phrase string) string {
	return strconv.Itoa(n) + "\x00" + phrase
}

func ensurePattern(stats map[string]*patternStat, key string, n int) *patternStat {
	stat := stats[key]
	if stat == nil {
		stat = &patternStat{N: n, Categories: make(map[string]int)}
		stats[key] = stat
	}
	return stat
}

func collectPatterns(stats map[string]*patternStat, minSupport, limit int, includeConcentration bool) []Pattern {
	out := make([]Pattern, 0, len(stats))
	for key, stat := range stats {
		if stat.Count < minSupport {
			continue
		}
		text := key[strings.IndexByte(key, '\x00')+1:]
		pattern := Pattern{Text: text, N: stat.N, Count: stat.Count, Examples: stat.Examples}
		if includeConcentration {
			for category, count := range stat.Categories {
				if count > pattern.DominantCount || (count == pattern.DominantCount && category < pattern.DominantCategory) {
					pattern.DominantCategory = category
					pattern.DominantCount = count
				}
			}
			if stat.Count > 0 {
				pattern.Concentration = float64(pattern.DominantCount) / float64(stat.Count)
			}
		} else {
			pattern.Concentration = 1
		}
		out = append(out, pattern)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		if out[i].Concentration != out[j].Concentration {
			return out[i].Concentration > out[j].Concentration
		}
		if out[i].N != out[j].N {
			return out[i].N > out[j].N
		}
		return out[i].Text < out[j].Text
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func appendExample(examples *[]string, value string, limit int) {
	for _, existing := range *examples {
		if existing == value {
			return
		}
	}
	if len(*examples) < limit {
		*examples = append(*examples, value)
	}
}

func tokenPositions(tokens []string, anchor string) []int {
	anchorTokens := strings.Fields(anchor)
	if len(anchorTokens) == 0 || len(tokens) < len(anchorTokens) {
		return nil
	}
	var positions []int
	for i := 0; i+len(anchorTokens) <= len(tokens); i++ {
		match := true
		for j := range anchorTokens {
			if tokens[i+j] != anchorTokens[j] {
				match = false
				break
			}
		}
		if match {
			positions = append(positions, i)
		}
	}
	return positions
}
