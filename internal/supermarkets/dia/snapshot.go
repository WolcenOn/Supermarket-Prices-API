package dia

import (
    "regexp"
    "strconv"
    "strings"
    "time"
)

var (
    skuMarkerRE = regexp.MustCompile(`sku_id\s*::\s*([0-9]+)`)
    dataMarkerRE = regexp.MustCompile(`(?i)^item\.data\s*::\s*`)
    moneyRE = regexp.MustCompile(`^([0-9]+,[0-9]{2})\s*€$`)
    discountRE = regexp.MustCompile(`^([0-9]+)%\s*dto\.$`)
    unitPriceRE = regexp.MustCompile(`^\(([0-9]+,[0-9]{2})\s*€/([^\)]+)\)$`)
)

// ParseRenderedSnapshot parses a plain-text snapshot of a DIA category/listing
// page. The snapshot format mirrors the semantic text exposed by the rendered
// public catalog and is used to lock down extraction rules without network I/O.
func ParseRenderedSnapshot(snapshot, postalCode string, observedAt time.Time) []RawProduct {
    lines := splitNonEmptyLines(snapshot)
    products := make([]RawProduct, 0)

    for i := 0; i < len(lines); i++ {
        match := skuMarkerRE.FindStringSubmatch(lines[i])
        if len(match) != 2 {
            continue
        }

        raw := RawProduct{
            ExternalID: match[1],
            PostalCode: strings.TrimSpace(postalCode),
            Available: true,
            ObservedAt: observedAt,
        }

        end := len(lines)
        for j := i + 1; j < len(lines); j++ {
            if skuMarkerRE.MatchString(lines[j]) {
                end = j
                break
            }
        }
        block := lines[i+1 : end]
        parseProductBlock(block, &raw)
        if raw.Name != "" && raw.RegularPrice > 0 {
            products = append(products, raw)
        }
        i = end - 1
    }

    return products
}

func parseProductBlock(lines []string, raw *RawProduct) {
    for _, line := range lines {
        normalized := normalizeVisibleWhitespace(line)
        normalized = strings.TrimSpace(dataMarkerRE.ReplaceAllString(normalized, ""))
        if normalized == "" {
            continue
        }
        lower := strings.ToLower(normalized)

        if strings.Contains(lower, "[button: agotado]") || lower == "agotado" {
            raw.Available = false
            continue
        }
        if strings.Contains(lower, "oferta club dia") {
            raw.PromotionType = "club"
            raw.PromotionLabel = "Oferta CLUB Dia"
            continue
        }
        if match := discountRE.FindStringSubmatch(lower); len(match) == 2 {
            raw.DiscountPct, _ = strconv.ParseFloat(match[1], 64)
            continue
        }
        if match := unitPriceRE.FindStringSubmatch(normalized); len(match) == 3 {
            raw.PricePerUnit = parseDecimal(match[1])
            raw.PriceUnit = strings.TrimSpace(match[2])
            continue
        }
        if match := moneyRE.FindStringSubmatch(normalized); len(match) == 2 {
            price := parseDecimal(match[1])
            if raw.RegularPrice == 0 {
                raw.RegularPrice = price
            } else if raw.PromotionalPrice == 0 {
                raw.PromotionalPrice = price
            }
            continue
        }

        if raw.Name == "" && isLikelyProductName(normalized) {
            raw.Name = normalized
        }
    }

    if raw.PromotionLabel == "" {
        for _, line := range lines {
            normalized := normalizeVisibleWhitespace(line)
            normalized = strings.TrimSpace(dataMarkerRE.ReplaceAllString(normalized, ""))
            upper := strings.ToUpper(normalized)
            if strings.Contains(upper, "DTO") && !discountRE.MatchString(strings.ToLower(normalized)) {
                raw.PromotionType = "multibuy"
                raw.PromotionLabel = normalized
                break
            }
        }
    }
}

func splitNonEmptyLines(value string) []string {
    rawLines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
    lines := make([]string, 0, len(rawLines))
    for _, line := range rawLines {
        line = normalizeVisibleWhitespace(line)
        if line != "" {
            lines = append(lines, line)
        }
    }
    return lines
}

func normalizeVisibleWhitespace(value string) string {
    replacer := strings.NewReplacer(
        "\u00a0", " ",
        "\u202f", " ",
        "\ufeff", "",
    )
    return strings.TrimSpace(replacer.Replace(value))
}

func isLikelyProductName(line string) bool {
    lower := strings.ToLower(line)
    if line == "" || strings.HasPrefix(line, "[") || strings.HasPrefix(line, "(") {
        return false
    }
    if strings.Contains(lower, "condition ::") || strings.Contains(lower, "oferta") ||
        strings.Contains(lower, "sin gluten") || strings.Contains(lower, "sin lactosa") ||
        strings.Contains(lower, "air fryer") || strings.Contains(lower, "novedad") ||
        strings.Contains(lower, "mejor valorado") || discountRE.MatchString(lower) || moneyRE.MatchString(line) {
        return false
    }
    return true
}

func parseDecimal(value string) float64 {
    value = strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
    parsed, _ := strconv.ParseFloat(value, 64)
    return parsed
}
