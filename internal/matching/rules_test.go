package matching

import (
    "testing"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func TestSuggestMatchesRawRiceVariants(t *testing.T) {
    cases := []struct {
        name string
        want string
    }{
        {"Arroz redondo SOS 1 Kg", "arroz_redondo"},
        {"Arroz extra Dia Arrozona 1 Kg", "arroz_extra"},
        {"Arroz vaporizado Dia Arrozona 1 Kg", "arroz_vaporizado"},
        {"Arroz basmati marca X 1 Kg", "arroz_basmati"},
        {"Arroz integral marca X 1 Kg", "arroz_integral"},
    }

    for _, tc := range cases {
        matches := Suggest(catalog.Product{SupermarketID: "dia", ExternalID: "sku", Name: tc.name})
        if len(matches) != 1 || matches[0].CanonicalIngredientID != tc.want {
            t.Fatalf("%q: got %#v, want %s", tc.name, matches, tc.want)
        }
    }
}

func TestSuggestRejectsPreparedRice(t *testing.T) {
    names := []string{
        "Arroz tres delicias Dia Al Punto 850 g",
        "Vasos de arroz integral Brillante 2 x 125 g",
        "Arroz de marisco Dia Al Punto 330 g",
        "Arroz de secreto ibérico y setas Selección de Dia 350 g",
    }
    for _, name := range names {
        if matches := Suggest(catalog.Product{SupermarketID: "dia", ExternalID: "sku", Name: name}); len(matches) != 0 {
            t.Fatalf("%q unexpectedly matched: %#v", name, matches)
        }
    }
}
