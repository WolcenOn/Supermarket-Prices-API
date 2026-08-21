package catalog

import "testing"

func TestInferPackageSize(t *testing.T) {
    tests := []struct {
        name       string
        product    string
        wantAmount float64
        wantUnit   string
        wantOK     bool
    }{
        {"kilogram", "Arroz redondo SOS 1 Kg", 1, "kg", true},
        {"grams", "Tomate frito 500 g", 500, "g", true},
        {"multipack", "Vasos de arroz Brillante 2 x 125 g", 250, "g", true},
        {"unicode multiply", "Leche 6 × 1,5 l", 9, "l", true},
        {"millilitres", "Caldo de pollo 750 ml", 750, "ml", true},
        {"no package", "Arroz a granel", 0, "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            gotAmount, gotUnit, gotOK := InferPackageSize(tt.product)
            if gotOK != tt.wantOK || gotAmount != tt.wantAmount || gotUnit != tt.wantUnit {
                t.Fatalf("InferPackageSize(%q) = (%v, %q, %v), want (%v, %q, %v)", tt.product, gotAmount, gotUnit, gotOK, tt.wantAmount, tt.wantUnit, tt.wantOK)
            }
        })
    }
}
