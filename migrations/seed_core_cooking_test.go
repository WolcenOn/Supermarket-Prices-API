package migrations

import (
    "strings"
    "testing"
)

func TestCoreCookingSeedsIncludeReviewedConcepts(t *testing.T) {
    canonicalSQL, err := Files.ReadFile("011_seed_core_cooking_canonical_ingredients.sql")
    if err != nil {
        t.Fatal(err)
    }
    aliasesSQL, err := Files.ReadFile("012_seed_verified_pack_aliases.sql")
    if err != nil {
        t.Fatal(err)
    }

    canonicalText := string(canonicalSQL)
    for _, required := range []string{
        "('huevo', 'Huevo'",
        "('pechuga_pollo', 'Pechuga de pollo'",
        "('aceite_oliva_virgen_extra', 'Aceite de oliva virgen extra'",
        "('caldo_verduras', 'Caldo de verduras'",
        "('merluza', 'Merluza'",
    } {
        if !strings.Contains(canonicalText, required) {
            t.Fatalf("missing reviewed canonical seed %q", required)
        }
    }

    aliasText := string(aliasesSQL)
    for _, required := range []string{
        "('espinaca', 'Espinacas'",
        "('calabacin', 'Calabacín pelado'",
        "('zanahoria', 'Zanahoria cocida'",
        "('huevo', 'Huevo cocido'",
        "('caldo_verduras', 'Caldo de verduras suave'",
    } {
        if !strings.Contains(aliasText, required) {
            t.Fatalf("missing reviewed verified alias %q", required)
        }
    }
}

func TestCoreCookingAliasesDoNotCollapseSemanticDistinctions(t *testing.T) {
    data, err := Files.ReadFile("012_seed_verified_pack_aliases.sql")
    if err != nil {
        t.Fatal(err)
    }
    text := string(data)

    // These terms intentionally remain unresolved/ambiguous until a specific
    // canonical policy exists. Do not trade precision for apparent coverage.
    for _, forbidden := range []string{
        "('tomate', 'Tomate seco'",
        "('tomate', 'Tomate triturado'",
        "('arroz_redondo', 'Arroz blanco'",
        "('pechuga_pollo', 'Pollo'",
        "('pechuga_pavo', 'Pavo'",
        "('leche_entera_sin_lactosa', 'Leche sin lactosa'",
        "('leche_semidesnatada_sin_lactosa', 'Leche sin lactosa'",
        "('leche_desnatada_sin_lactosa', 'Leche sin lactosa'",
    } {
        if strings.Contains(text, forbidden) {
            t.Fatalf("unsafe verified alias introduced: %q", forbidden)
        }
    }
}
