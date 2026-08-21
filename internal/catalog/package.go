package catalog

import (
    "regexp"
    "strconv"
    "strings"
)

var (
    multipackRE = regexp.MustCompile(`(?i)([0-9]+(?:[\.,][0-9]+)?)\s*[x×]\s*([0-9]+(?:[\.,][0-9]+)?)\s*(kg|g|l|ml|ud|uds|unidad|unidades)\b`)
    simplePackRE = regexp.MustCompile(`(?i)([0-9]+(?:[\.,][0-9]+)?)\s*(kg|g|l|ml|ud|uds|unidad|unidades)\s*$`)
)

// InferPackageSize extracts a total package amount and unit from a product name.
// For multipacks such as "2 x 125 g", the returned amount is the total package
// quantity (250 g), which is the useful value for basket/package calculations.
func InferPackageSize(name string) (amount float64, unit string, ok bool) {
    name = strings.TrimSpace(name)
    if name == "" {
        return 0, "", false
    }

    if match := multipackRE.FindStringSubmatch(name); len(match) == 4 {
        count, countOK := parsePackageNumber(match[1])
        each, eachOK := parsePackageNumber(match[2])
        if !countOK || !eachOK || count <= 0 || each <= 0 {
            return 0, "", false
        }
        return count * each, NormalizePackageUnit(match[3]), true
    }

    if match := simplePackRE.FindStringSubmatch(name); len(match) == 3 {
        amount, amountOK := parsePackageNumber(match[1])
        if !amountOK || amount <= 0 {
            return 0, "", false
        }
        return amount, NormalizePackageUnit(match[2]), true
    }

    return 0, "", false
}

func NormalizePackageUnit(unit string) string {
    switch strings.ToLower(strings.TrimSpace(unit)) {
    case "kilo", "kilogramo", "kilogramos", "kg":
        return "kg"
    case "litro", "litros", "l":
        return "l"
    case "gramo", "gramos", "g":
        return "g"
    case "mililitro", "mililitros", "ml":
        return "ml"
    case "unidad", "unidades", "ud", "uds":
        return "unit"
    default:
        return strings.ToLower(strings.TrimSpace(unit))
    }
}

func parsePackageNumber(value string) (float64, bool) {
    value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
    parsed, err := strconv.ParseFloat(value, 64)
    return parsed, err == nil
}
