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

func TestCoreCookingV2SeedsReviewedShoppingConcepts(t *testing.T) {
    canonicalSQL, err := Files.ReadFile("013_seed_core_cooking_canonical_ingredients_v2.sql")
    if err != nil {
        t.Fatal(err)
    }
    aliasesSQL, err := Files.ReadFile("014_seed_verified_pack_aliases_v2.sql")
    if err != nil {
        t.Fatal(err)
    }

    canonicalText := string(canonicalSQL)
    for _, required := range []string{
        "('bebida_avena_sin_azucar', 'Bebida de avena sin azúcar'",
        "('jamon_cocido', 'Jamón cocido'",
        "('atun_al_natural', 'Atún al natural'",
        "('tomate_cherry', 'Tomate cherry'",
        "('cebolla_morada', 'Cebolla morada'",
        "('tomate_seco', 'Tomate seco'",
        "('semillas_calabaza', 'Semillas de calabaza'",
    } {
        if !strings.Contains(canonicalText, required) {
            t.Fatalf("missing reviewed v2 canonical seed %q", required)
        }
    }

    aliasText := string(aliasesSQL)
    for _, required := range []string{
        "('jamon_cocido', 'Jamón york'",
        "('tomate_cherry', 'Tomates cherry'",
        "('pimiento_rojo', 'Pimiento rojo crudo'",
        "('queso_parmesano', 'Parmesano'",
    } {
        if !strings.Contains(aliasText, required) {
            t.Fatalf("missing reviewed v2 verified alias %q", required)
        }
    }
}

func TestCoreCookingV2DoesNotPretendCookedDryStaplesAreRawPurchaseQuantities(t *testing.T) {
    data, err := Files.ReadFile("014_seed_verified_pack_aliases_v2.sql")
    if err != nil {
        t.Fatal(err)
    }
    text := string(data)

    for _, forbidden := range []string{
        "Pasta integral cocida",
        "Arroz integral cocido",
        "Quinoa cocida",
        "Cuscús integral cocido",
    } {
        if strings.Contains(text, forbidden) {
            t.Fatalf("cooked dry staple must not be verified as a raw purchase alias: %q", forbidden)
        }
    }
}
