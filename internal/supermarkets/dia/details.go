package dia

import (
    "context"
    "fmt"
    "html"
    "io"
    "net/http"
    "net/url"
    "regexp"
    "strconv"
    "strings"
)

type NutritionFacts struct {
    BasisAmount       float64 `json:"basisAmount,omitempty"`
    BasisUnit         string  `json:"basisUnit,omitempty"`
    EnergyKJ          float64 `json:"energyKJ,omitempty"`
    EnergyKcal        float64 `json:"energyKcal,omitempty"`
    FatG              float64 `json:"fatG,omitempty"`
    SaturatedFatG     float64 `json:"saturatedFatG,omitempty"`
    CarbohydratesG    float64 `json:"carbohydratesG,omitempty"`
    SugarsG           float64 `json:"sugarsG,omitempty"`
    FiberG            float64 `json:"fiberG,omitempty"`
    ProteinG          float64 `json:"proteinG,omitempty"`
    SaltG             float64 `json:"saltG,omitempty"`
}

type ProductDetails struct {
    ExternalID            string          `json:"externalId,omitempty"`
    Name                  string          `json:"name,omitempty"`
    EAN                   string          `json:"ean,omitempty"`
    SourceURL             string          `json:"sourceUrl"`
    DescriptionText       string          `json:"descriptionText,omitempty"`
    SourceIngredientsBlock string         `json:"sourceIngredientsBlock,omitempty"`
    IngredientsText       string          `json:"ingredientsText,omitempty"`
    ResponsibleText       string          `json:"responsibleText,omitempty"`
    NutritionSource       string          `json:"nutritionSource,omitempty"`
    Nutrition             *NutritionFacts `json:"nutrition,omitempty"`
}

var (
    h1RE         = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
    gtinRE       = regexp.MustCompile(`(?is)(?:ean13|ean|gtin14|gtin13|gtin|barcode)[^0-9]{0,24}([0-9]{8,14})`)
    basisRE      = regexp.MustCompile(`(?i)valores?\s+por\s+([0-9]+(?:[.,][0-9]+)?)\s*(g|ml)\b`)
    numberUnitRE = regexp.MustCompile(`(?i)([0-9]+(?:[.,][0-9]+)?)\s*(kj|kcal|g)\b`)
)

func (s *HTTPSource) FetchProductDetails(ctx context.Context, rawURL string) (ProductDetails, error) {
    s.ensureDefaults()

    parsed, err := url.Parse(strings.TrimSpace(rawURL))
    if err != nil || !isPublicDIAProductURL(parsed) {
        return ProductDetails{}, fmt.Errorf("dia: product detail URL must be a public https://www.dia.es/.../p/<sku> URL")
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
    if err != nil {
        return ProductDetails{}, fmt.Errorf("dia: build product detail request: %w", err)
    }
    s.setBrowserHeaders(req)

    resp, err := s.Client.Do(req)
    if err != nil {
        return ProductDetails{}, fmt.Errorf("dia: fetch product detail: %w", err)
    }
    defer resp.Body.Close()
    if !isPublicDIAProductURL(resp.Request.URL) {
        return ProductDetails{}, fmt.Errorf("dia: product detail redirected outside an allowed DIA product URL")
    }
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return ProductDetails{}, fmt.Errorf("dia: fetch product detail: unexpected status %d", resp.StatusCode)
    }

    body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
    if err != nil {
        return ProductDetails{}, fmt.Errorf("dia: read product detail: %w", err)
    }

    details := ParseProductDetailHTML(string(body))
    details.SourceURL = resp.Request.URL.String()
    details.ExternalID = productIDFromURL(resp.Request.URL)
    return details, nil
}

func ParseProductDetailHTML(document string) ProductDetails {
    semantic := HTMLToSemanticText(document)
    lines := strings.Split(semantic, "\n")
    sourceIngredients := extractSemanticSection(lines, "ingredientes", []string{"conservacion y utilizacion", "informacion del responsable"})

    details := ProductDetails{
        Name:                   extractH1(document),
        EAN:                    extractValidGTIN(document),
        DescriptionText:        extractDescriptionBeforeHeading(lines, "ingredientes"),
        SourceIngredientsBlock: sourceIngredients,
        ResponsibleText:        extractSemanticSection(lines, "informacion del responsable", nil),
    }
    if looksLikeIngredientDeclaration(sourceIngredients) {
        details.IngredientsText = sourceIngredients
    }
    if nutrition, ok := parseNutrition(lines); ok {
        details.Nutrition = &nutrition
        details.NutritionSource = "dia_product_page"
    }
    return details
}

func parseNutrition(lines []string) (NutritionFacts, bool) {
    var facts NutritionFacts
    found := false

    for i, line := range lines {
        normalized := normalizeDetailLabel(line)
        if match := basisRE.FindStringSubmatch(normalized); len(match) == 3 {
            if value, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64); err == nil {
                facts.BasisAmount = value
                facts.BasisUnit = strings.ToLower(match[2])
                found = true
            }
        }

        switch normalized {
        case "valor energetico":
            for j := i + 1; j < len(lines) && j <= i+4; j++ {
                value, unit, ok := parseNumberUnit(lines[j])
                if !ok {
                    continue
                }
                if unit == "kj" {
                    facts.EnergyKJ = value
                    found = true
                } else if unit == "kcal" {
                    facts.EnergyKcal = value
                    found = true
                }
            }
        case "grasas":
            facts.FatG, found = nextGramValue(lines, i, facts.FatG, found)
        case "de las cuales saturadas":
            facts.SaturatedFatG, found = nextGramValue(lines, i, facts.SaturatedFatG, found)
        case "hidratos de carbono":
            facts.CarbohydratesG, found = nextGramValue(lines, i, facts.CarbohydratesG, found)
        case "de los cuales azucares":
            facts.SugarsG, found = nextGramValue(lines, i, facts.SugarsG, found)
        case "fibra", "fibra alimentaria":
            facts.FiberG, found = nextGramValue(lines, i, facts.FiberG, found)
        case "proteinas":
            facts.ProteinG, found = nextGramValue(lines, i, facts.ProteinG, found)
        case "sal":
            facts.SaltG, found = nextGramValue(lines, i, facts.SaltG, found)
        }
    }

    return facts, found
}

func nextGramValue(lines []string, index int, current float64, found bool) (float64, bool) {
    for j := index + 1; j < len(lines) && j <= index+3; j++ {
        value, unit, ok := parseNumberUnit(lines[j])
        if ok && unit == "g" {
            return value, true
        }
    }
    return current, found
}

func parseNumberUnit(value string) (float64, string, bool) {
    match := numberUnitRE.FindStringSubmatch(strings.ToLower(strings.TrimSpace(value)))
    if len(match) != 3 {
        return 0, "", false
    }
    number, err := strconv.ParseFloat(strings.ReplaceAll(match[1], ",", "."), 64)
    if err != nil {
        return 0, "", false
    }
    return number, strings.ToLower(match[2]), true
}

func extractH1(document string) string {
    match := h1RE.FindStringSubmatch(document)
    if len(match) != 2 {
        return ""
    }
    value := tagRE.ReplaceAllString(match[1], " ")
    value = html.UnescapeString(value)
    return strings.Join(strings.Fields(value), " ")
}

func extractDescriptionBeforeHeading(lines []string, heading string) string {
    target := normalizeDetailLabel(heading)
    for i, line := range lines {
        if normalizeDetailLabel(line) != target {
            continue
        }
        for j := i - 1; j >= 0 && j >= i-4; j-- {
            value := strings.TrimSpace(lines[j])
            normalized := normalizeDetailLabel(value)
            if value == "" || normalized == "/" || isNutritionLabel(normalized) {
                continue
            }
            if _, _, ok := parseNumberUnit(value); ok {
                continue
            }
            if basisRE.MatchString(normalized) {
                continue
            }
            return value
        }
        break
    }
    return ""
}

func isNutritionLabel(value string) bool {
    switch value {
    case "valor nutricional", "valor energetico", "grasas", "de las cuales saturadas", "hidratos de carbono", "de los cuales azucares", "fibra", "fibra alimentaria", "proteinas", "sal":
        return true
    default:
        return false
    }
}

func looksLikeIngredientDeclaration(value string) bool {
    normalized := normalizeDetailLabel(value)
    if normalized == "" {
        return false
    }
    nonDeclarations := []string{
        "tipo de ",
        "tipo del ",
        "tipo producto ",
        "variedad ",
        "categoria ",
        "calibre ",
        "origen ",
        "formato ",
    }
    for _, prefix := range nonDeclarations {
        if strings.HasPrefix(normalized, prefix) {
            return false
        }
    }
    return true
}

func extractSemanticSection(lines []string, heading string, stops []string) string {
    heading = normalizeDetailLabel(heading)
    stopSet := make(map[string]struct{}, len(stops))
    for _, stop := range stops {
        stopSet[normalizeDetailLabel(stop)] = struct{}{}
    }

    start := -1
    for i, line := range lines {
        if normalizeDetailLabel(line) == heading {
            start = i + 1
            break
        }
    }
    if start < 0 {
        return ""
    }

    values := make([]string, 0)
    for i := start; i < len(lines); i++ {
        normalized := normalizeDetailLabel(lines[i])
        if _, stop := stopSet[normalized]; stop {
            break
        }
        if strings.Contains(normalized, "€") || strings.EqualFold(normalized, "anadir") || strings.EqualFold(normalized, "te recomendamos") {
            break
        }
        value := strings.TrimSpace(lines[i])
        if value != "" {
            values = append(values, value)
        }
        if len(values) >= 8 {
            break
        }
    }
    return strings.Join(values, " ")
}

func extractValidGTIN(document string) string {
    for _, match := range gtinRE.FindAllStringSubmatch(document, -1) {
        if len(match) != 2 {
            continue
        }
        candidate := strings.TrimSpace(match[1])
        if validGTIN(candidate) {
            return candidate
        }
    }
    return ""
}

func validGTIN(value string) bool {
    switch len(value) {
    case 8, 12, 13, 14:
    default:
        return false
    }
    sum := 0
    for i := len(value) - 2; i >= 0; i-- {
        digit := int(value[i] - '0')
        if digit < 0 || digit > 9 {
            return false
        }
        positionFromRight := len(value) - 1 - i
        if positionFromRight%2 == 1 {
            sum += digit * 3
        } else {
            sum += digit
        }
    }
    check := (10 - (sum % 10)) % 10
    last := int(value[len(value)-1] - '0')
    return last >= 0 && last <= 9 && check == last
}

func productIDFromURL(parsed *url.URL) string {
    if parsed == nil {
        return ""
    }
    parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
    for i := 0; i+1 < len(parts); i++ {
        if parts[i] == "p" {
            return strings.TrimSpace(parts[i+1])
        }
    }
    return ""
}

func normalizeDetailLabel(value string) string {
    value = strings.ToLower(strings.TrimSpace(value))
    replacer := strings.NewReplacer(
        "á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
        "\u00a0", " ", ":", "",
    )
    value = replacer.Replace(value)
    return strings.Join(strings.Fields(value), " ")
}
