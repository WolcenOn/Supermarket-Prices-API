package catalogscan

import (
	"strings"

	canonicaltext "github.com/WolcenOn/Supermarket-Prices-API/internal/canonical"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/matching"
)

const (
	DecisionAutoMatchCandidate = "auto_match_candidate"
	DecisionNoCanonical        = "no_canonical"
	DecisionReview             = "review"
	DecisionNonRecipe          = "non_recipe"
	DecisionInvalid            = "invalid"
)

var simpleProduceSourceCategories = map[string]struct{}{
	"l2022": {},
	"l2023": {},
	"l2024": {},
	"l2027": {},
	"l2028": {},
	"l2029": {},
	"l2031": {},
	"l2181": {},
}

var reviewOnlyProduceTerms = []string{
	"ensalada",
	"microondas",
	"mezcla",
	"mix ",
}

type Issue struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type MatchCandidate struct {
	CanonicalIngredientID string  `json:"canonicalIngredientId"`
	Score                 float64 `json:"score"`
	Source                string  `json:"source"`
}

type Analysis struct {
	Decision        string           `json:"decision"`
	MatchCandidates []MatchCandidate `json:"matchCandidates,omitempty"`
	Issues          []Issue          `json:"issues,omitempty"`
}

type Item struct {
	Product  catalog.Product `json:"product"`
	Analysis Analysis        `json:"analysis"`
}

func Analyze(product catalog.Product) Analysis {
	issues := validate(product)
	if hasInvalidIssue(issues) {
		return Analysis{Decision: DecisionInvalid, Issues: issues}
	}

	if product.ClassificationStatus == "classified" && !product.RecipeCompatible {
		return Analysis{Decision: DecisionNonRecipe, Issues: issues}
	}

	matches := matching.Suggest(product)
	candidates := make([]MatchCandidate, 0, len(matches))
	for _, match := range matches {
		candidates = append(candidates, MatchCandidate{
			CanonicalIngredientID: match.CanonicalIngredientID,
			Score:                 match.Score,
			Source:                match.Source,
		})
	}

	decision := DecisionReview
	if len(candidates) == 1 && candidates[0].Score >= 0.95 {
		decision = DecisionAutoMatchCandidate
	} else if len(candidates) == 0 && isSimpleProduceWithoutCanonical(product) {
		decision = DecisionNoCanonical
	}

	return Analysis{
		Decision:        decision,
		MatchCandidates: candidates,
		Issues:          issues,
	}
}

func AnalyzeAll(products []catalog.Product) []Item {
	items := make([]Item, 0, len(products))
	for _, product := range products {
		items = append(items, Item{Product: product, Analysis: Analyze(product)})
	}
	return items
}

func isSimpleProduceWithoutCanonical(product catalog.Product) bool {
	if product.ClassificationStatus != "classified" ||
		!product.RecipeCompatible ||
		product.ItemType != "food_ingredient" {
		return false
	}
	if product.NormalizedCategory != "food.produce.vegetable" &&
		product.NormalizedCategory != "food.produce.mushroom" {
		return false
	}
	categoryID := strings.ToLower(strings.TrimSpace(product.SourceCategoryID))
	if _, ok := simpleProduceSourceCategories[categoryID]; !ok {
		return false
	}
	name := canonicaltext.NormalizeText(product.Name)
	for _, term := range reviewOnlyProduceTerms {
		if strings.Contains(name, term) {
			return false
		}
	}
	return true
}

func validate(product catalog.Product) []Issue {
	issues := make([]Issue, 0)
	if strings.TrimSpace(product.SupermarketID) == "" {
		issues = append(issues, invalid("missing_supermarket_id", "supermarket id is required"))
	}
	if strings.TrimSpace(product.ExternalID) == "" {
		issues = append(issues, invalid("missing_external_id", "external product id is required"))
	}
	if strings.TrimSpace(product.Name) == "" {
		issues = append(issues, invalid("missing_name", "product name is required"))
	}
	if product.Price < 0 {
		issues = append(issues, invalid("negative_price", "price cannot be negative"))
	}
	if product.PackageAmount < 0 {
		issues = append(issues, invalid("negative_package_amount", "package amount cannot be negative"))
	}
	if product.PackageAmount > 0 && strings.TrimSpace(product.PackageUnit) == "" {
		issues = append(issues, review("package_unit_missing", "package amount is present but package unit is missing"))
	}
	if product.PricePerUnit < 0 {
		issues = append(issues, invalid("negative_price_per_unit", "price per unit cannot be negative"))
	}
	if product.PricePerUnit > 0 && strings.TrimSpace(product.PriceUnit) == "" {
		issues = append(issues, review("price_unit_missing", "price per unit is present but price unit is missing"))
	}
	if product.ClassificationStatus == "" {
		issues = append(issues, review("classification_missing", "product has not been classified"))
	}
	return issues
}

func invalid(code, message string) Issue {
	return Issue{Code: code, Severity: "invalid", Message: message}
}

func review(code, message string) Issue {
	return Issue{Code: code, Severity: "review", Message: message}
}

func hasInvalidIssue(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Severity == "invalid" {
			return true
		}
	}
	return false
}
