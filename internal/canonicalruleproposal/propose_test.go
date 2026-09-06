package canonicalruleproposal

import (
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalanalysis"
)

func TestProposeBuildsSpecificRuleAndPreparedVetoes(t *testing.T) {
	analysis := canonicalanalysis.Report{
		ProductsTotal: 20,
		ProductsUsed: 20,
		GlobalPatterns: []canonicalanalysis.Pattern{
			{
				Text: "jamon cocido", N: 2, Count: 8,
				DominantCategory: "charcuteria/jamon-cocido/c/L2001",
				DominantCount: 8, Concentration: 1,
				Examples: []string{"Jamón cocido extra 150 g"},
			},
			{Text: "jamon", N: 1, Count: 15, DominantCategory: "charcuteria/jamon-cocido/c/L2001", DominantCount: 8, Concentration: 0.5333},
		},
		Categories: []canonicalanalysis.CategoryReport{
			{
				Path: "charcuteria/jamon-cocido/c/L2001",
				Root: "charcuteria",
				Products: 8,
			},
			{
				Path: "platos-preparados-y-pizzas/pizzas/c/L3000",
				Root: "platos-preparados-y-pizzas",
				Products: 4,
				TopPatterns: []canonicalanalysis.Pattern{
					{Text: "pizza jamon", N: 2, Count: 3},
				},
			},
		},
		AnchorContexts: []canonicalanalysis.AnchorContext{
			{
				Anchor: "jamon",
				Products: 15,
				Categories: map[string]int{
					"charcuteria/jamon-cocido/c/L2001": 8,
					"platos-preparados-y-pizzas/pizzas/c/L3000": 4,
				},
			},
		},
	}

	report := Propose(analysis, Options{MinSupport: 3, MinConcentration: 0.70})
	if len(report.Families) != 1 {
		t.Fatalf("families = %d, want 1", len(report.Families))
	}
	family := report.Families[0]
	if len(family.Proposals) != 1 {
		t.Fatalf("proposals = %d, want 1", len(family.Proposals))
	}
	proposal := family.Proposals[0]
	if proposal.Pattern != "jamon cocido" {
		t.Fatalf("pattern = %q", proposal.Pattern)
	}
	if proposal.SuggestedConceptID != "jamon_cocido" {
		t.Fatalf("suggestedConceptId = %q", proposal.SuggestedConceptID)
	}
	if len(proposal.IdentityModifiers) != 1 || proposal.IdentityModifiers[0] != "cocido" {
		t.Fatalf("identityModifiers = %#v", proposal.IdentityModifiers)
	}
	if len(proposal.CategoryBoosts) != 1 || proposal.CategoryBoosts[0] != "charcuteria/jamon-cocido/c/L2001" {
		t.Fatalf("categoryBoosts = %#v", proposal.CategoryBoosts)
	}
	if len(family.CategoryVetoes) != 1 || family.CategoryVetoes[0] != "platos-preparados-y-pizzas/pizzas/c/L3000" {
		t.Fatalf("categoryVetoes = %#v", family.CategoryVetoes)
	}
	if len(family.PreparedVetoes) != 1 || family.PreparedVetoes[0] != "pizza jamon" {
		t.Fatalf("preparedVetoes = %#v", family.PreparedVetoes)
	}
}

func TestProposeIgnoresGenericAndDispersedPatterns(t *testing.T) {
	analysis := canonicalanalysis.Report{
		GlobalPatterns: []canonicalanalysis.Pattern{
			{Text: "pollo", N: 1, Count: 20, Concentration: 0.4},
			{Text: "pollo caldo", N: 2, Count: 7, DominantCategory: "conservas-caldos-y-cremas/caldos/c/L1", Concentration: 0.55},
		},
		AnchorContexts: []canonicalanalysis.AnchorContext{{Anchor: "pollo", Products: 20}},
	}

	report := Propose(analysis, Options{MinSupport: 3, MinConcentration: 0.70})
	if len(report.Families) != 1 {
		t.Fatalf("families = %d, want 1", len(report.Families))
	}
	if got := len(report.Families[0].Proposals); got != 0 {
		t.Fatalf("proposals = %d, want 0", got)
	}
}
