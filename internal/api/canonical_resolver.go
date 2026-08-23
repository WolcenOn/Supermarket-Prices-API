package api

import (
    "net/http"
    "strings"

    "github.com/WolcenOn/Supermarket-Prices-API/internal/catalog"
)

func (h *Handler) resolveCanonicalIngredient(w http.ResponseWriter, r *http.Request) {
    resolverStore, ok := h.store.(catalog.CanonicalResolverStore)
    if !ok {
        writeJSON(w, http.StatusServiceUnavailable, map[string]any{
            "error":   "canonical_resolver_unavailable",
            "message": "canonical ingredient resolver is not configured",
        })
        return
    }

    query := strings.TrimSpace(r.URL.Query().Get("q"))
    if query == "" {
        writeJSON(w, http.StatusBadRequest, map[string]any{
            "error":   "missing_query",
            "message": "query parameter q is required",
        })
        return
    }

    resolution, err := resolverStore.ResolveCanonicalIngredient(r.Context(), query)
    if err != nil {
        writeStoreError(w)
        return
    }
    writeJSON(w, http.StatusOK, resolution)
}
