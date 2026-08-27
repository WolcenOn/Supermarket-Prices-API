package catalog

import (
    "fmt"
    "math"
    "strings"
)

type PurchaseQuote struct {
    Product          Product `json:"product"`
    RequiredAmount  float64 `json:"requiredAmount"`
    RequiredUnit    string  `json:"requiredUnit"`
    PackageCount    int     `json:"packageCount,omitempty"`
    PurchasedAmount float64 `json:"purchasedAmount"`
    PurchasedUnit   string  `json:"purchasedUnit"`
    WasteAmount     float64 `json:"wasteAmount,omitempty"`
    TotalCost       float64 `json:"totalCost"`
    Approximate     bool    `json:"approximate,omitempty"`
}

// QuotePurchase calculates the real checkout cost for a required ingredient
// amount. Packaged products are rounded up to whole packages. Variable-weight
// products are costed directly from price per unit and their purchased amount
// and total cost are estimates until the final weighed amount is known.
func QuotePurchase(product Product, requiredAmount float64, requiredUnit string) (PurchaseQuote, error) {
    requiredUnit = normalizeMeasureUnit(requiredUnit)
    if requiredAmount <= 0 {
        return PurchaseQuote{}, fmt.Errorf("required amount must be positive")
    }
    if requiredUnit == "" {
        return PurchaseQuote{}, fmt.Errorf("required unit is required")
    }

    quote := PurchaseQuote{
        Product:         product,
        RequiredAmount: requiredAmount,
        RequiredUnit:   requiredUnit,
    }

    if product.VariableWeight {
        priceUnit := normalizeMeasureUnit(product.PriceUnit)
        if product.PricePerUnit <= 0 || priceUnit == "" {
            return PurchaseQuote{}, fmt.Errorf("variable-weight product requires price per unit")
        }
        amountInPriceUnit, err := convertMeasure(requiredAmount, requiredUnit, priceUnit)
        if err != nil {
            return PurchaseQuote{}, err
        }
        quote.PurchasedAmount = roundQuantity(amountInPriceUnit)
        quote.PurchasedUnit = priceUnit
        quote.TotalCost = roundMoney(amountInPriceUnit * product.PricePerUnit)
        quote.Approximate = true
        return quote, nil
    }

    packageUnit := normalizeMeasureUnit(product.PackageUnit)
    if product.PackageAmount <= 0 || packageUnit == "" {
        return PurchaseQuote{}, fmt.Errorf("packaged product requires package amount and unit")
    }
    if product.Price <= 0 {
        return PurchaseQuote{}, fmt.Errorf("product price must be positive")
    }

    requiredInPackageUnit, err := convertMeasure(requiredAmount, requiredUnit, packageUnit)
    if err != nil {
        return PurchaseQuote{}, err
    }

    packages := int(math.Ceil(requiredInPackageUnit / product.PackageAmount))
    purchased := float64(packages) * product.PackageAmount
    quote.PackageCount = packages
    quote.PurchasedAmount = roundQuantity(purchased)
    quote.PurchasedUnit = packageUnit
    quote.WasteAmount = roundQuantity(math.Max(0, purchased-requiredInPackageUnit))
    quote.TotalCost = roundMoney(float64(packages) * product.Price)
    quote.Approximate = isApproximateWholeUnit(product)
    return quote, nil
}

func isApproximateWholeUnit(product Product) bool {
    if product.VariableWeight || product.PackageAmount <= 0 || normalizeMeasureUnit(product.PackageUnit) == "" {
        return false
    }
    lowerName := strings.ToLower(strings.TrimSpace(product.Name))
    if strings.Contains(lowerName, "aprox") {
        return true
    }
    if strings.Contains(lowerName, "unidad") && product.PricePerUnit > 0 {
        priceUnit := normalizeMeasureUnit(product.PriceUnit)
        return priceUnit == "kg" || priceUnit == "g"
    }
    return false
}

func convertMeasure(amount float64, from, to string) (float64, error) {
    from = normalizeMeasureUnit(from)
    to = normalizeMeasureUnit(to)
    if from == to {
        return amount, nil
    }

    switch {
    case from == "g" && to == "kg":
        return amount / 1000, nil
    case from == "kg" && to == "g":
        return amount * 1000, nil
    case from == "ml" && to == "l":
        return amount / 1000, nil
    case from == "l" && to == "ml":
        return amount * 1000, nil
    default:
        return 0, fmt.Errorf("incompatible units %q and %q", from, to)
    }
}

func normalizeMeasureUnit(unit string) string {
    switch strings.ToLower(strings.TrimSpace(unit)) {
    case "g", "gramo", "gramos":
        return "g"
    case "kg", "kilo", "kilos", "kilogramo", "kilogramos":
        return "kg"
    case "ml", "mililitro", "mililitros":
        return "ml"
    case "l", "litro", "litros":
        return "l"
    case "unit", "unidad", "unidades", "ud", "uds":
        return "unit"
    default:
        return strings.ToLower(strings.TrimSpace(unit))
    }
}

func roundMoney(value float64) float64 {
    return math.Round(value*100) / 100
}

func roundQuantity(value float64) float64 {
    return math.Round(value*1000) / 1000
}
