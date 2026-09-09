package canonicalanalysis

import "testing"

func TestAnalyzeKeepsSemanticContextsSeparatedByCategory(t *testing.T) {
	products := []Product{
		product("1", "Jamón cocido extra 150 g", "L2001", "Jamón cocido", "charcuteria/jamon-cocido/c/L2001"),
		product("2", "Jamón cocido reducido en sal 200 g", "L2001", "Jamón cocido", "charcuteria/jamon-cocido/c/L2001"),
		product("3", "Croquetas de jamón 500 g", "L2135", "Croquetas y rebozados", "congelados-y-helados/croquetas-y-rebozados/c/L2135"),
	}

	report := Analyze(products, Options{FoodOnly: true, MinSupport: 1, TopGlobal: 100, TopPerCategory: 20, Anchors: []string{"jamón"}})
	if report.ProductsUsed != 3 {
		t.Fatalf("products used = %d, want 3", report.ProductsUsed)
	}
	anchor := findAnchor(t, report, "jamon")
	if anchor.Products != 3 {
		t.Fatalf("jamon products = %d, want 3", anchor.Products)
	}
	if anchor.Categories["charcuteria/jamon-cocido/c/L2001"] != 2 {
		t.Fatalf("jamon charcuteria count = %d, want 2", anchor.Categories["charcuteria/jamon-cocido/c/L2001"])
	}
	if anchor.Categories["congelados-y-helados/croquetas-y-rebozados/c/L2135"] != 1 {
		t.Fatalf("jamon prepared count = %d, want 1", anchor.Categories["congelados-y-helados/croquetas-y-rebozados/c/L2135"])
	}
}

func TestAnalyzeUsesWholeTokensForAnchors(t *testing.T) {
	products := []Product{
		product("1", "Pollo entero 2 kg", "L2202", "Pollo", "carnes/pollo/c/L2202"),
		product("2", "Repollo cortado 400 g", "L2030", "Ensaladas", "verduras/ensaladas-y-verduras-preparadas/c/L2030"),
	}
	report := Analyze(products, Options{FoodOnly: true, MinSupport: 1, Anchors: []string{"pollo"}})
	anchor := findAnchor(t, report, "pollo")
	if anchor.Products != 1 {
		t.Fatalf("pollo products = %d, want 1", anchor.Products)
	}
}

func TestAnalyzeCountsPatternOncePerProduct(t *testing.T) {
	products := []Product{
		product("1", "Arroz arroz redondo 1 kg", "L2042", "Arroz", "arroz-pastas-y-legumbres/arroz/c/L2042"),
	}
	report := Analyze(products, Options{FoodOnly: true, MinSupport: 1, TopGlobal: 100})
	for _, pattern := range report.GlobalPatterns {
		if pattern.Text == "arroz" {
			if pattern.Count != 1 {
				t.Fatalf("arroz count = %d, want 1 product support", pattern.Count)
			}
			return
		}
	}
	t.Fatal("arroz pattern not found")
}

func TestAnalyzeTopGlobalZeroKeepsAllSupportedPatterns(t *testing.T) {
	products := []Product{
		product("1", "Jamón cocido extra", "L2001", "Jamón cocido", "charcuteria/jamon-cocido/c/L2001"),
		product("2", "Jamón cocido reducido sal", "L2001", "Jamón cocido", "charcuteria/jamon-cocido/c/L2001"),
	}

	complete := Analyze(products, Options{FoodOnly: true, MinSupport: 1, TopGlobal: 0})
	limited := Analyze(products, Options{FoodOnly: true, MinSupport: 1, TopGlobal: 1})
	if len(complete.GlobalPatterns) <= 1 {
		t.Fatalf("unbounded global patterns = %d, want more than 1", len(complete.GlobalPatterns))
	}
	if len(limited.GlobalPatterns) != 1 {
		t.Fatalf("limited global patterns = %d, want 1", len(limited.GlobalPatterns))
	}
	if !hasPattern(complete.GlobalPatterns, "jamon cocido") {
		t.Fatal("unbounded global patterns should retain specific jamon cocido pattern")
	}
}

func TestAnalyzeFoodOnlyExcludesNonFoodRoots(t *testing.T) {
	products := []Product{
		product("1", "Champú hidratante 750 ml", "L2144", "Champu", "cabello-y-perfumeria/champu/c/L2144"),
		product("2", "Leche entera 1 l", "L2051", "Leche", "huevos-leche-y-mantequilla/leche/c/L2051"),
	}
	report := Analyze(products, Options{FoodOnly: true, MinSupport: 1})
	if report.ProductsUsed != 1 {
		t.Fatalf("products used = %d, want 1", report.ProductsUsed)
	}
}

func product(id, name, categoryID, categoryName, categoryPath string) Product {
	var p Product
	p.ExternalID = id
	p.Name = name
	p.Category.ID = categoryID
	p.Category.Name = categoryName
	p.Category.Path = categoryPath
	return p
}

func findAnchor(t *testing.T, report Report, anchor string) AnchorContext {
	t.Helper()
	for _, item := range report.AnchorContexts {
		if item.Anchor == anchor {
			return item
		}
	}
	t.Fatalf("anchor %q not found", anchor)
	return AnchorContext{}
}

func hasPattern(patterns []Pattern, text string) bool {
	for _, pattern := range patterns {
		if pattern.Text == text {
			return true
		}
	}
	return false
}
