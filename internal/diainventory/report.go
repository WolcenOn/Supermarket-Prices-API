package diainventory

import (
    "context"
    "database/sql"
    "fmt"
    "sort"
    "strings"
    "time"
)

type Product struct {
    ID                 string    `json:"id"`
    ExternalID         string    `json:"externalId"`
    Name               string    `json:"name"`
    Brand              string    `json:"brand,omitempty"`
    EAN                string    `json:"ean,omitempty"`
    SourceURL          string    `json:"sourceUrl,omitempty"`
    SourceCategoryID   string    `json:"sourceCategoryId,omitempty"`
    SourceCategoryName string    `json:"sourceCategoryName,omitempty"`
    SourceCategoryPath string    `json:"sourceCategoryPath,omitempty"`
    PackageAmount      float64   `json:"packageAmount,omitempty"`
    PackageUnit        string    `json:"packageUnit,omitempty"`
    VariableWeight     bool      `json:"variableWeight"`
    Price              float64   `json:"price"`
    PricePerUnit       float64   `json:"pricePerUnit,omitempty"`
    PriceUnit          string    `json:"priceUnit,omitempty"`
    PostalCode         string    `json:"postalCode,omitempty"`
    Available          bool      `json:"available"`
    ObservedAt         time.Time `json:"observedAt"`
    ObservationCount   int       `json:"observationCount"`
}

type Category struct {
    ID           string `json:"id,omitempty"`
    Name         string `json:"name,omitempty"`
    Path         string `json:"path,omitempty"`
    ProductCount int    `json:"productCount"`
}

type Report struct {
    SupermarketID    string     `json:"supermarketId"`
    PostalCode       string     `json:"postalCode,omitempty"`
    ProductCount     int        `json:"productCount"`
    CategoryCount    int        `json:"categoryCount"`
    ObservationCount int        `json:"observationCount"`
    LatestObservedAt *time.Time `json:"latestObservedAt,omitempty"`
    Categories       []Category `json:"categories"`
    Items            []Product  `json:"items,omitempty"`
}

func Load(ctx context.Context, db *sql.DB, postalCode string, includeItems bool) (Report, error) {
    if db == nil {
        return Report{}, fmt.Errorf("DIA inventory: database is required")
    }
    postalCode = strings.TrimSpace(postalCode)

    rows, err := db.QueryContext(ctx, `
        SELECT
            sp.id::text,
            sp.external_id,
            sp.name,
            COALESCE(sp.brand, ''),
            COALESCE(sp.ean, ''),
            COALESCE(sp.source_url, ''),
            COALESCE(sp.source_category_id, ''),
            COALESCE(sp.source_category_name, ''),
            COALESCE(sp.source_category_path, ''),
            COALESCE(sp.package_amount, 0),
            COALESCE(sp.package_unit, ''),
            sp.variable_weight,
            latest.price,
            COALESCE(latest.price_per_unit, 0),
            COALESCE(latest.price_unit, ''),
            COALESCE(latest.postal_code, ''),
            latest.available,
            latest.observed_at,
            (
                SELECT COUNT(*)
                FROM price_observations history
                WHERE history.supermarket_product_id = sp.id
                  AND ($1 = '' OR history.postal_code = $1 OR history.postal_code IS NULL)
            )::int
        FROM supermarket_products sp
        JOIN LATERAL (
            SELECT observation.*
            FROM price_observations observation
            WHERE observation.supermarket_product_id = sp.id
              AND ($1 = '' OR observation.postal_code = $1 OR observation.postal_code IS NULL)
            ORDER BY
                CASE WHEN $1 <> '' AND observation.postal_code = $1 THEN 0 ELSE 1 END,
                observation.observed_at DESC
            LIMIT 1
        ) latest ON TRUE
        WHERE sp.supermarket_id = 'dia'
        ORDER BY COALESCE(sp.source_category_path, ''), sp.name, sp.external_id
    `, postalCode)
    if err != nil {
        return Report{}, fmt.Errorf("DIA inventory: query products: %w", err)
    }
    defer rows.Close()

    products := make([]Product, 0)
    categories := make(map[string]Category)
    observationCount := 0
    var latestObservedAt *time.Time

    for rows.Next() {
        var item Product
        if err := rows.Scan(
            &item.ID,
            &item.ExternalID,
            &item.Name,
            &item.Brand,
            &item.EAN,
            &item.SourceURL,
            &item.SourceCategoryID,
            &item.SourceCategoryName,
            &item.SourceCategoryPath,
            &item.PackageAmount,
            &item.PackageUnit,
            &item.VariableWeight,
            &item.Price,
            &item.PricePerUnit,
            &item.PriceUnit,
            &item.PostalCode,
            &item.Available,
            &item.ObservedAt,
            &item.ObservationCount,
        ); err != nil {
            return Report{}, fmt.Errorf("DIA inventory: scan product: %w", err)
        }

        observationCount += item.ObservationCount
        if latestObservedAt == nil || item.ObservedAt.After(*latestObservedAt) {
            observed := item.ObservedAt
            latestObservedAt = &observed
        }

        key := item.SourceCategoryID + "|" + item.SourceCategoryPath
        category := categories[key]
        category.ID = item.SourceCategoryID
        category.Name = item.SourceCategoryName
        category.Path = item.SourceCategoryPath
        category.ProductCount++
        categories[key] = category

        if includeItems {
            products = append(products, item)
        }
    }
    if err := rows.Err(); err != nil {
        return Report{}, fmt.Errorf("DIA inventory: iterate products: %w", err)
    }

    categoryItems := make([]Category, 0, len(categories))
    for _, category := range categories {
        categoryItems = append(categoryItems, category)
    }
    sort.Slice(categoryItems, func(i, j int) bool {
        if categoryItems[i].Path == categoryItems[j].Path {
            return categoryItems[i].ID < categoryItems[j].ID
        }
        return categoryItems[i].Path < categoryItems[j].Path
    })

    productCount := 0
    for _, category := range categoryItems {
        productCount += category.ProductCount
    }

    return Report{
        SupermarketID:    "dia",
        PostalCode:       postalCode,
        ProductCount:     productCount,
        CategoryCount:    len(categoryItems),
        ObservationCount: observationCount,
        LatestObservedAt: latestObservedAt,
        Categories:       categoryItems,
        Items:            products,
    }, nil
}
