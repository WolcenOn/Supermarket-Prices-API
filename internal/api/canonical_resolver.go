package api

import (
    "net/http"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

const maxCanonicalResolveBatch = 100

func (h *Handler) resolveCanonicalIngredient(w http.ResponseWriter, r *http.Request) {
    resolverStore, ok := h.store.(catalog.CanonicalResolverStore)
    if !ok {
        writeJSON(w, http.StatusServiceUnavailable, map[string]any{
            "error":   "canonical_resolver_unavailable",
            "message": "canonical ingredient resolver is not configured",
        })
        return
    }

    rawQueries := r.URL.Query()["q"]
    queries := make([]string, 0, len(rawQueries))
    for _, raw := range rawQueries {
        query := strings.TrimSpace(raw)
        if query != "" {
            queries = append(queries, query)
        }
    }
    if len(queries) == 0 {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error":   "missing_query",
            "message": "query parameter q is required",
        })
        return
    }
    if len(queries) > maxCanonicalResolveBatch {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error":   "too_many_queries",
            "message": "at most 100 q query parameters are allowed",
        })
        return
    }

    if len(queries) == 1 {
        resolution, err := resolverStore.ResolveCanonicalIngredient(r.Context(), queries[0])
        if err != nil {
            writeStoreError(w)
            return
        }
        writeJSON(w, http.StatusOK, resolution)
        return
    }

    items := make([]catalog.CanonicalResolution, 0, len(queries))
    for _, query := range queries {
        resolution, err := resolverStore.ResolveCanonicalIngredient(r.Context(), query)
        if err != nil {
            writeStoreError(w)
            return
        }
        items = append(items, resolution)
    }
    writeJSON(w, http.StatusOK, map[string]any{
        "count": len(items),
        "items": items,
    })
}
