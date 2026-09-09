package canonicalsemantics

import (
	"testing"

	"github.com/WolcenOn/Supermarket-Prices-API/internal/canonicalruleproposal"
)

func TestAuditSeparatesIngredientFromContext(t *testing.T) {
	input := canonicalruleproposal.Report{ProductsTotal: 20, ProductsUsed: 20, Families: []canonicalruleproposal.FamilyReport{{
		Family: "jamon", Products: 10,
		Proposals: []canonicalruleproposal.Proposal{
			{Family: "jamon", Pattern: "jamon cocido", CategoryBoosts: []string{"charcuteria/jamon-cocido/c/L2001"}, EvidenceScore: 0.95},
			{Family: "jamon", Pattern: "pizza jamon", CategoryBoosts: []string{"platos-preparados-y-pizzas/pizzas-refrigeradas/c/L2101"}, EvidenceScore: 0.90},
		},
	}}}
	report := Audit(input)
	if got := report.Summary[RoleIngredientCandidate]; got != 1 { t.Fatalf("ingredient candidates = %d, want 1", got) }
	if got := report.Summary[RolePreparedContext]; got != 1 { t.Fatalf("prepared contexts = %d, want 1", got) }
}

func TestAuditUsesContextTermsBeforeCategory(t *testing.T) {
	proposal := canonicalruleproposal.Proposal{Family: "pollo", Pattern: "sabor pollo", CategoryBoosts: []string{"carnes/pollo/c/L2202"}}
	got := auditProposal("pollo", proposal)
	if got.Role != RoleContextOnly { t.Fatalf("role = %s, want %s", got.Role, RoleContextOnly) }
}

func TestAuditRecognizesProcessedIngredientCategory(t *testing.T) {
	proposal := canonicalruleproposal.Proposal{Family: "tomate", Pattern: "tomate triturado", CategoryBoosts: []string{"aceites-salsas-y-especias/salsas-de-tomate-y-pasta/c/L2208"}}
	got := auditProposal("tomate", proposal)
	if got.Role != RoleProcessedIngredient { t.Fatalf("role = %s, want %s", got.Role, RoleProcessedIngredient) }
}

func TestAuditKeepsUnknownFamiliesForReview(t *testing.T) {
	proposal := canonicalruleproposal.Proposal{Family: "desconocido", Pattern: "algo concreto"}
	got := auditProposal("desconocido", proposal)
	if got.Role != RoleReview { t.Fatalf("role = %s, want %s", got.Role, RoleReview) }
}
