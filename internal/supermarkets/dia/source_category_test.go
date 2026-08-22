package dia

import "testing"

func TestSourceCategoryFromURL(t *testing.T) {
    id, name, path := sourceCategoryFromURL("https://www.dia.es/arroz-pastas-y-legumbres/c/L106")

    if id != "L106" {
        t.Fatalf("id = %q, want L106", id)
    }
    if name != "Arroz pastas y legumbres" {
        t.Fatalf("name = %q, want %q", name, "Arroz pastas y legumbres")
    }
    if path != "arroz-pastas-y-legumbres/c/L106" {
        t.Fatalf("path = %q", path)
    }
}
