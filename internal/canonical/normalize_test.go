package canonical

import "testing"

func TestNormalizeTextKeepsSemanticWords(t *testing.T) {
    got := NormalizeText("  Arroz de grano REDONDO  ")
    if got != "arroz de grano redondo" {
        t.Fatalf("unexpected normalized text %q", got)
    }
}

func TestNormalizeTextRemovesAccentsAndPunctuation(t *testing.T) {
    got := NormalizeText("Aceite de oliva virgen extra (AOVE) / 1 L")
    if got != "aceite de oliva virgen extra aove 1 l" {
        t.Fatalf("unexpected normalized text %q", got)
    }
}

func TestNormalizeTextTreatsIdentifierSeparatorsAsSpaces(t *testing.T) {
    got := NormalizeText("leche_semidesnatada-sin_lactosa")
    if got != "leche semidesnatada sin lactosa" {
        t.Fatalf("unexpected normalized text %q", got)
    }
}
