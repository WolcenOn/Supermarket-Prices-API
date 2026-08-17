package api

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const Version = "0.1.0"

type Handler struct {
    store catalog.Store
}

func NewHandler(store catalog.Store) *Handler {
    return &Handler{store: store}
}

func (h *Handler) Routes() http.Handler {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /health", h.health)
    mux.HandleFunc("GET /api/v1/version", h.version)
    mux.HandleFunc("GET /api/v1/supermarkets", h.supermarkets)
    mux.HandleFunc("GET /api/v1/products/search", h.searchProducts)
    return mux
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
