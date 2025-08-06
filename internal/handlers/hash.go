package handlers

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/jirapanel/internal/config"
)

// HashHandler returns an HTTP handler that responds with a hash of either the full config
// or a specific cell layout, based on the requested path parameter.
func HashHandler(
	cfg config.DashboardConfig,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		switch id {
		case "config":
			// Hash the entire dashboard config
			hash, err := hashAny(cfg)
			if err != nil {
				http.Error(w, "failed to compute hash for config", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(hash)) // nolint:errcheck
			return

		default:
			// Attempt to parse the id as a cell index
			idx, err := strconv.Atoi(id)
			if err != nil {
				http.Error(w, "invalid cell id", http.StatusBadRequest)
				return
			}

			cell, err := cfg.GetLayoutByIndex(idx)
			if err != nil {
				http.Error(w, "cell not found", http.StatusNotFound)
				return
			}

			hash, err := hashAny(cell)
			if err != nil {
				logger.Error("hash computation failed", "id", id, "error", err)
				http.Error(w, "failed to compute hash", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(hash)) // nolint:errcheck
		}
	}
}

// hashAny serializes the given value and returns its FNV-1a 64-bit hash as a hex string.
func hashAny(a any) (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("failed to serialize: %w", err)
	}

	h := fnv.New64a()
	if _, err := h.Write(data); err != nil {
		return "", fmt.Errorf("failed to hash data: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum64()), nil
}
