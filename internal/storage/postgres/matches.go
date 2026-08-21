package postgres

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/matching"
)

func SaveIngredientMatches(ctx context.Context, db *sql.DB, matches []matching.Match) error {
    if len(matches) == 0 {
        return nil
    }
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("postgres matches: begin: %w", err)
    }
    defer tx.Rollback()

    for _, match := range matches {
        _, err := tx.ExecContext(ctx, `
            INSERT INTO ingredient_product_matches (
                canonical_ingredient_id,
                supermarket_product_id,
                match_status,
                match_score,
                match_source,
                updated_at
            )
            SELECT
                $1,
                sp.id,
                $4,
                $5,
                $6,
                NOW()
            FROM supermarket_products sp
            WHERE sp.supermarket_id = $2
              AND sp.external_id = $3
            ON CONFLICT (canonical_ingredient_id, supermarket_product_id)
            DO UPDATE SET
                match_status = CASE
                    WHEN ingredient_product_matches.match_status IN ('confirmed', 'rejected') THEN ingredient_product_matches.match_status
                    ELSE EXCLUDED.match_status
                END,
                match_score = CASE
                    WHEN ingredient_product_matches.match_status IN ('confirmed', 'rejected') THEN ingredient_product_matches.match_score
                    ELSE EXCLUDED.match_score
                END,
                match_source = CASE
                    WHEN ingredient_product_matches.match_status IN ('confirmed', 'rejected') THEN ingredient_product_matches.match_source
                    ELSE EXCLUDED.match_source
                END,
                updated_at = NOW()
        `, match.CanonicalIngredientID, match.SupermarketID, match.ExternalID, match.Status, match.Score, match.Source)
        if err != nil {
            return fmt.Errorf("postgres matches: save %s/%s -> %s: %w", match.SupermarketID, match.ExternalID, match.CanonicalIngredientID, err)
        }
    }

    if err := tx.Commit(); err != nil {
        return fmt.Errorf("postgres matches: commit: %w", err)
    }
    return nil
}
