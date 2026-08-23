package postgres

import (
    "context"
    "database/sql"
    "fmt"
    "strings"

    canonicaltext "github.com/WolcenOn/Supermarket-Prices-API/internal/canonical"
    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func (s *CatalogStore) ResolveCanonicalIngredient(ctx context.Context, query string) (catalog.CanonicalResolution, error) {
    if s == nil || s.db == nil {
        return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: database is required")
    }

    query = strings.TrimSpace(query)
    normalized := canonicaltext.NormalizeText(query)
    result := catalog.CanonicalResolution{
        Query:           query,
        NormalizedQuery: normalized,
        Status:          "unresolved",
        Candidates:      []catalog.CanonicalResolutionCandidate{},
    }
    if normalized == "" {
        return result, nil
    }

    // Canonical names and IDs are authoritative by definition. We normalize
    // them in application code so accents and separators follow exactly the
    // same rules as aliases without requiring PostgreSQL extensions.
    rows, err := s.db.QueryContext(ctx, `
        SELECT id, name, COALESCE(category, ''), COALESCE(subtype, ''), COALESCE(default_unit, '')
        FROM canonical_ingredients
        ORDER BY id
        LIMIT 5000
    `)
    if err != nil {
        return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: resolve canonical names: %w", err)
    }
    for rows.Next() {
        var ingredient catalog.CanonicalIngredient
        if err := rows.Scan(&ingredient.ID, &ingredient.Name, &ingredient.Category, &ingredient.Subtype, &ingredient.DefaultUnit); err != nil {
            rows.Close()
            return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: scan canonical name: %w", err)
        }
        if canonicaltext.NormalizeText(ingredient.Name) == normalized || canonicaltext.NormalizeText(ingredient.ID) == normalized {
            rows.Close()
            result.Status = "verified"
            result.Candidates = []catalog.CanonicalResolutionCandidate{{
                Ingredient: ingredient,
                MatchType:  "canonical_name",
                Confidence: 1,
            }}
            return result, nil
        }
    }
    if err := rows.Err(); err != nil {
        rows.Close()
        return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: iterate canonical names: %w", err)
    }
    rows.Close()

    aliasRows, err := s.db.QueryContext(ctx, `
        SELECT
            a.id,
            a.canonical_ingredient_id,
            a.alias,
            a.normalized_alias,
            a.status,
            a.confidence,
            a.decision_source,
            a.verification_note,
            ci.id,
            ci.name,
            COALESCE(ci.category, ''),
            COALESCE(ci.subtype, ''),
            COALESCE(ci.default_unit, '')
        FROM canonical_ingredient_aliases a
        JOIN canonical_ingredients ci ON ci.id = a.canonical_ingredient_id
        WHERE a.normalized_alias = $1
          AND a.status IN ('verified', 'suggested')
        ORDER BY
            CASE WHEN a.status = 'verified' THEN 0 ELSE 1 END,
            a.confidence DESC NULLS LAST,
            ci.name
        LIMIT 20
    `, normalized)
    if err != nil {
        return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: resolve aliases: %w", err)
    }
    defer aliasRows.Close()

    verified := make([]catalog.CanonicalResolutionCandidate, 0, 1)
    suggested := make([]catalog.CanonicalResolutionCandidate, 0)
    for aliasRows.Next() {
        var alias catalog.CanonicalIngredientAlias
        var ingredient catalog.CanonicalIngredient
        var confidence sql.NullFloat64
        var note sql.NullString
        if err := aliasRows.Scan(
            &alias.ID,
            &alias.CanonicalIngredientID,
            &alias.Alias,
            &alias.NormalizedAlias,
            &alias.Status,
            &confidence,
            &alias.DecisionSource,
            &note,
            &ingredient.ID,
            &ingredient.Name,
            &ingredient.Category,
            &ingredient.Subtype,
            &ingredient.DefaultUnit,
        ); err != nil {
            return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: scan alias resolution: %w", err)
        }
        if confidence.Valid {
            alias.Confidence = confidence.Float64
        }
        if note.Valid {
            alias.VerificationNote = note.String
        }
        candidate := catalog.CanonicalResolutionCandidate{
            Ingredient: ingredient,
            Alias:      &alias,
            MatchType:  "alias",
            Confidence: alias.Confidence,
        }
        if alias.Status == "verified" {
            verified = append(verified, candidate)
        } else {
            suggested = append(suggested, candidate)
        }
    }
    if err := aliasRows.Err(); err != nil {
        return catalog.CanonicalResolution{}, fmt.Errorf("postgres catalog: iterate alias resolution: %w", err)
    }

    if len(verified) > 0 {
        result.Status = "verified"
        result.Candidates = verified
        return result, nil
    }
    if len(suggested) > 0 {
        result.Status = "suggested"
        result.Candidates = suggested
    }
    return result, nil
}
