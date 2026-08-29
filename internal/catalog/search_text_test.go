package catalog

import "testing"

func TestSearchTextMatchesWholeTokens(t *testing.T) {
    if !SearchTextMatches("Pechuga de pollo familiar", "pollo") {
        t.Fatal("expected pollo token to match")
    }
    if SearchTextMatches("Repollo liso", "pollo") {
        t.Fatal("pollo must not match inside repollo")
    }
}

func TestSearchTextMatchesIgnoresSpanishAccents(t *testing.T) {
    if !SearchTextMatches("Champú familiar hidratante", "champu") {
        t.Fatal("expected accent-insensitive champu match")
    }
    if !SearchTextMatches("Champu familiar hidratante", "champú") {
        t.Fatal("expected accent-insensitive champú match")
    }
}

func TestSearchTextMatchesRequiresEveryMultiwordTerm(t *testing.T) {
    if !SearchTextMatches("Pechuga fresca de pollo", "pollo pechuga") {
        t.Fatal("expected all query tokens to match regardless of order")
    }
    if SearchTextMatches("Pechuga fresca de pavo", "pollo pechuga") {
        t.Fatal("missing pollo token must not match")
    }
}
