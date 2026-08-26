package api

import (
    "encoding/json"
    "net/http"
    "sort"
    "strconv"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const Version = "0.3.3"

type Handler struct {
    store           catalog.Store
    ingredientStore catalog.IngredientStore
    nutritionStore  catalog.NutritionStore
}

func NewHandler(store catalog.Store) *Handler {
    handler := &Handler{store: store}
    if ingredientStore, ok := store.(catalog.IngredientStore); ok {
        handler.ingredientStore = ingredientStore
    }
    if nutritionStore, ok := store.(catalog.NutritionStore); ok {
        handler.nutritionStore = nutritionStore
    }
    return handler
}

func (h *Handler) Routes() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", h.health)
    mux.HandleFunc("GET /api/v1/version", h.version)
    mux.HandleFunc("GET /api/v1/supermarkets", h.supermarkets)
    mux.HandleFunc("GET /api/v1/products/search", h.searchProducts)
    mux.HandleFunc("GET /api/v1/products/{id}/nutrition", h.productNutrition)
    mux.HandleFunc("GET /api/v1/ingredients", h.ingredients)
    mux.HandleFunc("GET /api/v1/ingredients/search", h.searchIngredients)
    mux.HandleFunc("GET /api/v1/ingredients/resolve", h.resolveCanonicalIngredient)
    mux.HandleFunc("GET /api/v1/ingredients/{id}/products", h.ingredientProducts)
    mux.HandleFunc("GET /api/v1/ingredients/{id}/quote", h.ingredientQuote)
    return withPublicReadCORS(mux)
}

func withPublicReadCORS(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type")
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusNoContent)
            return
        }
        next.ServeHTTP(w, r)
    })
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) version(w http.ResponseWriter, _ *http.Request) {
    writeJSON(w, http.StatusOK, map[string]any{"version": Version})
}

func (h *Handler) supermarkets(w http.ResponseWriter, r *http.Request) {
    items, err := h.store.Supermarkets(r.Context())
    if err != nil {
        writeStoreError(w)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) searchProducts(w http.ResponseWriter, r *http.Request) {
    query := strings.TrimSpace(r.URL.Query().Get("q"))
    if query == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "missing_query",
            "message": "query parameter q is required",
        })
        return
    }

    postalCode := strings.TrimSpace(r.URL.Query().Get("postalCode"))
    products, err := h.store.Search(r.Context(), catalog.SearchParams{Query: query, PostalCode: postalCode})
    if err != nil {
        writeStoreError(w)
        return
    }

    writeJSON(w, http.StatusOK, map[string]any{
        "query": query,
        "postalCode": postalCode,
        "count": len(products),
        "items": products,
    })
}

func (h *Handler) productNutrition(w http.ResponseWriter, r *http.Request) {
    if !h.requireNutritionStore(w) {
        return
    }
    productID := strings.TrimSpace(r.PathValue("id"))
    if productID == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "missing_product_id",
            "message": "product id is required",
        })
        return
    }

    items, err := h.nutritionStore.ProductNutrition(r.Context(), productID)
    if err != nil {
        writeStoreError(w)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "productId": productID,
        "count": len(items),
        "items": items,
    })
}

func (h *Handler) ingredients(w http.ResponseWriter, r *http.Request) {
    if !h.requireIngredientStore(w) {
        return
    }
    items, err := h.ingredientStore.Ingredients(r.Context())
    if err != nil {
        writeStoreError(w)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "count": len(items),
        "items": items,
    })
}

func (h *Handler) searchIngredients(w http.ResponseWriter, r *http.Request) {
    if !h.requireIngredientStore(w) {
        return
    }
    query := strings.TrimSpace(r.URL.Query().Get("q"))
    if query == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "missing_query",
            "message": "query parameter q is required",
        })
        return
    }
    items, err := h.ingredientStore.SearchIngredients(r.Context(), query)
    if err != nil {
        writeStoreError(w)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "query": query,
        "count": len(items),
        "items": items,
    })
}

func (h *Handler) ingredientProducts(w http.ResponseWriter, r *http.Request) {
    if !h.requireIngredientStore(w) {
        return
    }
    ingredientID := strings.TrimSpace(r.PathValue("id"))
    if ingredientID == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "missing_ingredient_id",
            "message": "ingredient id is required",
        })
        return
    }
    postalCode := strings.TrimSpace(r.URL.Query().Get("postalCode"))
    items, err := h.ingredientStore.IngredientProducts(r.Context(), ingredientID, postalCode)
    if err != nil {
        writeStoreError(w)
        return
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "ingredientId": ingredientID,
        "postalCode": postalCode,
        "count": len(items),
        "items": items,
    })
}

func (h *Handler) ingredientQuote(w http.ResponseWriter, r *http.Request) {
    if !h.requireIngredientStore(w) {
        return
    }

    ingredientID := strings.TrimSpace(r.PathValue("id"))
    if ingredientID == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "missing_ingredient_id",
            "message": "ingredient id is required",
        })
        return
    }

    amountText := strings.TrimSpace(r.URL.Query().Get("amount"))
    amount, err := strconv.ParseFloat(amountText, 64)
    if err != nil || amount <= 0 {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "invalid_amount",
            "message": "query parameter amount must be a positive number",
        })
        return
    }

    unit := strings.TrimSpace(r.URL.Query().Get("unit"))
    if unit == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error": "missing_unit",
            "message": "query parameter unit is required",
        })
        return
    }

    postalCode := strings.TrimSpace(r.URL.Query().Get("postalCode"))
    products, err := h.ingredientStore.IngredientProducts(r.Context(), ingredientID, postalCode)
    if err != nil {
        writeStoreError(w)
        return
    }

    quotes := make([]catalog.PurchaseQuote, 0, len(products))
    skipped := 0
    for _, item := range products {
        if !item.Product.Available {
            skipped++
            continue
        }
        quote, quoteErr := catalog.QuotePurchase(item.Product, amount, unit)
        if quoteErr != nil {
            skipped++
            continue
        }
        quotes = append(quotes, quote)
    }

    sort.SliceStable(quotes, func(i, j int) bool {
        if quotes[i].TotalCost == quotes[j].TotalCost {
            return quotes[i].Product.Name < quotes[j].Product.Name
        }
        return quotes[i].TotalCost < quotes[j].TotalCost
    })

    writeJSON(w, http.StatusOK, map[string]any{
        "ingredientId": ingredientID,
        "postalCode": postalCode,
        "requiredAmount": amount,
        "requiredUnit": unit,
        "count": len(quotes),
        "skipped": skipped,
        "items": quotes,
    })
}

func (h *Handler) requireIngredientStore(w http.ResponseWriter) bool {
    if h.ingredientStore != nil {
        return true
    }
    writeJSON(w, http.StatusServiceUnavailable, map[string]any{
        "error": "ingredient_catalog_unavailable",
        "message": "canonical ingredient catalog is not configured",
    })
    return false
}

func (h *Handler) requireNutritionStore(w http.ResponseWriter) bool {
    if h.nutritionStore != nil {
        return true
    }
    writeJSON(w, http.StatusServiceUnavailable, map[string]any{
        "error": "nutrition_catalog_unavailable",
        "message": "product nutrition catalog is not configured",
    })
    return false
}

func writeStoreError(w http.ResponseWriter) {
    writeJSON(w, http.StatusInternalServerError, map[string]any{
        "error": "catalog_unavailable",
        "message": "catalog data is temporarily unavailable",
    })
}

func writeJSON(w http.ResponseWriter, status int, value any) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(value)
}
