package canonicalsemantics

import (
	"sort"
	"strings"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalruleproposal"
	"github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

type Role string

const (
	RoleIngredientCandidate Role = "ingredient_candidate"
	RoleProcessedIngredient Role = "processed_ingredient_candidate"
	RolePreparedContext Role = "prepared_context"
	RoleContextOnly Role = "context_only"
	RoleReview Role = "review"
)

type AuditedProposal struct {
	canonicalruleproposal.Proposal
	Role   Role     `json:"role"`
	Reason []string `json:"reason"`
}

type FamilyReport struct {
	Family    string            `json:"family"`
	Products  int               `json:"products"`
	Proposals []AuditedProposal `json:"proposals"`
}

type Report struct {
	ProductsTotal int            `json:"productsTotal"`
	ProductsUsed  int            `json:"productsUsed"`
	Families      []FamilyReport `json:"families"`
	Summary       map[Role]int   `json:"summary"`
}

type profile struct {
	IngredientPrefixes []string
	ProcessedPrefixes  []string
	ContextPrefixes    []string
	ContextTerms       []string
}

var profiles = map[string]profile{
	"jamon": {IngredientPrefixes: []string{"charcuteria/jamon-cocido/", "charcuteria/jamon-serrano/"}, ContextPrefixes: []string{"platos-preparados-y-pizzas/", "congelados-y-helados/", "aperitivos-y-frutos-secos/", "galletas-cereales-y-mermeladas/"}},
	"pollo": {IngredientPrefixes: []string{"carnes/pollo/", "carnes/hamburguesas-carne-picada-y-albondigas/"}, ProcessedPrefixes: []string{"charcuteria/pavo-y-pollo/"}, ContextPrefixes: []string{"conservas-caldos-y-cremas/caldos-y-sopas/", "arroz-pastas-y-legumbres/noodles/", "platos-preparados-y-pizzas/", "congelados-y-helados/"}, ContextTerms: []string{"sabor", "caldo", "sopa", "noodles"}},
	"pavo": {IngredientPrefixes: []string{"carnes/pavo/"}, ProcessedPrefixes: []string{"charcuteria/pavo-y-pollo/"}},
	"cerdo": {IngredientPrefixes: []string{"carnes/cerdo/", "carnes/hamburguesas-carne-picada-y-albondigas/"}, ProcessedPrefixes: []string{"charcuteria/"}},
	"vacuno": {IngredientPrefixes: []string{"carnes/vacuno/", "carnes/hamburguesas-carne-picada-y-albondigas/"}},
	"tomate": {IngredientPrefixes: []string{"verduras/tomates-pimientos-y-pepinos/"}, ProcessedPrefixes: []string{"aceites-salsas-y-especias/salsas-de-tomate-y-pasta/"}},
	"arroz": {IngredientPrefixes: []string{"arroz-pastas-y-legumbres/arroz/"}, ContextPrefixes: []string{"galletas-cereales-y-mermeladas/tortitas/", "galletas-cereales-y-mermeladas/cereales", "congelados-y-helados/"}, ContextTerms: []string{"tortitas", "cereales"}},
	"leche": {IngredientPrefixes: []string{"huevos-leche-y-mantequilla/leche/", "huevos-leche-y-mantequilla/leche-sin-lactosa-y-enriquecidas/"}, ProcessedPrefixes: []string{"huevos-leche-y-mantequilla/leche-condensada-y-evaporada/"}, ContextPrefixes: []string{"chocolates-y-golosinas/", "zumos-y-smoothies/fruta-y-leche/", "galletas-cereales-y-mermeladas/"}, ContextTerms: []string{"chocolate", "fruta", "barritas"}},
	"queso": {IngredientPrefixes: []string{"quesos/"}, ContextPrefixes: []string{"aperitivos-y-frutos-secos/", "platos-preparados-y-pizzas/"}, ContextTerms: []string{"sabor"}},
	"huevo": {IngredientPrefixes: []string{"huevos-leche-y-mantequilla/huevos/"}, ContextPrefixes: []string{"yogures-y-postres/", "bolleria-reposteria-y-azucar/", "platos-preparados-y-pizzas/"}},
	"atun": {IngredientPrefixes: []string{"conservas-caldos-y-cremas/atun-y-bonito/", "pescados-y-mariscos/"}, ContextPrefixes: []string{"platos-preparados-y-pizzas/", "congelados-y-helados/", "panaderia/"}},
	"merluza": {IngredientPrefixes: []string{"pescados-y-mariscos/congelado/", "congelados-y-helados/pescado-y-marisco/"}, ProcessedPrefixes: []string{"pescados-y-mariscos/rebozado/"}},
	"patata": {IngredientPrefixes: []string{"verduras/patatas-y-zanahorias/"}, ContextPrefixes: []string{"aperitivos-y-frutos-secos/", "platos-preparados-y-pizzas/"}},
	"cebolla": {IngredientPrefixes: []string{"verduras/ajos-cebollas-y-puerros/"}, ContextPrefixes: []string{"platos-preparados-y-pizzas/"}},
	"pimiento": {IngredientPrefixes: []string{"verduras/tomates-pimientos-y-pepinos/"}, ContextPrefixes: []string{"platos-preparados-y-pizzas/"}},
	"aceite": {IngredientPrefixes: []string{"aceites-salsas-y-especias/aceites/"}, ContextPrefixes: []string{"conservas-caldos-y-cremas/", "pescados-y-mariscos/"}},
	"yogur": {IngredientPrefixes: []string{"yogures-y-postres/yogures-"}, ContextPrefixes: []string{"congelados-y-helados/", "galletas-cereales-y-mermeladas/"}},
}

func Audit(input canonicalruleproposal.Report) Report {
	out := Report{ProductsTotal: input.ProductsTotal, ProductsUsed: input.ProductsUsed, Summary: make(map[Role]int)}
	for _, family := range input.Families {
		fr := FamilyReport{Family: family.Family, Products: family.Products}
		for _, proposal := range family.Proposals {
			audited := auditProposal(family.Family, proposal)
			fr.Proposals = append(fr.Proposals, audited)
			out.Summary[audited.Role]++
		}
		sort.Slice(fr.Proposals, func(i, j int) bool {
			if fr.Proposals[i].Role != fr.Proposals[j].Role { return fr.Proposals[i].Role < fr.Proposals[j].Role }
			if fr.Proposals[i].EvidenceScore != fr.Proposals[j].EvidenceScore { return fr.Proposals[i].EvidenceScore > fr.Proposals[j].EvidenceScore }
			return fr.Proposals[i].Pattern < fr.Proposals[j].Pattern
		})
		out.Families = append(out.Families, fr)
	}
	return out
}

func auditProposal(family string, proposal canonicalruleproposal.Proposal) AuditedProposal {
	family = catalog.NormalizeSearchText(family)
	p, ok := profiles[family]
	if !ok { return AuditedProposal{Proposal: proposal, Role: RoleReview, Reason: []string{"no_family_profile"}} }
	category := ""
	if len(proposal.CategoryBoosts) > 0 { category = proposal.CategoryBoosts[0] }
	pattern := catalog.NormalizeSearchText(proposal.Pattern)
	if containsAnyTerm(pattern, p.ContextTerms) { return AuditedProposal{Proposal: proposal, Role: RoleContextOnly, Reason: []string{"context_term"}} }
	if hasPrefix(category, p.IngredientPrefixes) { return AuditedProposal{Proposal: proposal, Role: RoleIngredientCandidate, Reason: []string{"ingredient_category"}} }
	if hasPrefix(category, p.ProcessedPrefixes) { return AuditedProposal{Proposal: proposal, Role: RoleProcessedIngredient, Reason: []string{"processed_ingredient_category"}} }
	if hasPrefix(category, p.ContextPrefixes) || hasPrefix(category, proposal.CategoryVetoes) { return AuditedProposal{Proposal: proposal, Role: RolePreparedContext, Reason: []string{"context_category"}} }
	return AuditedProposal{Proposal: proposal, Role: RoleReview, Reason: []string{"unclassified_category"}}
}

func hasPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes { if strings.HasPrefix(value, prefix) { return true } }
	return false
}

func containsAnyTerm(value string, terms []string) bool {
	fields := strings.Fields(value)
	set := make(map[string]struct{}, len(fields))
	for _, field := range fields { set[field] = struct{}{} }
	for _, term := range terms { if _, ok := set[term]; ok { return true } }
	return false
}
