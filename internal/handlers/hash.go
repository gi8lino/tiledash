package handlers

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/hash"
	"github.com/gi8lino/tiledash/internal/render"
)

// HashHandler returns an HTTP handler that responds with a hash of either the full config
// or the current data for a specific tile, based on the requested path parameter.
func HashHandler(
	cfg config.DashboardConfig,
	renderer *render.TileRenderer,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		switch id {
		case "config":
			// Hash the entire dashboard config
			h, err := hash.Any(cfg)
			if err != nil {
				http.Error(w, "failed to compute hash for config", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(h)) // nolint:errcheck
			return

		default:
			// Attempt to parse the id as a tile index
			idx, err := strconv.Atoi(id)
			if err != nil || renderer == nil {
				http.Error(w, "invalid tile id", http.StatusBadRequest)
				return
			}

			result, status, renderErr := renderer.RenderTile(r.Context(), idx)
			if renderErr != nil {
				if status == http.StatusNotFound || status == http.StatusBadRequest {
					http.Error(w, "invalid tile id", status)
					return
				}
				logger.Error("hash computation failed", "id", id, "status", status, "error", renderErr.Error())
				http.Error(w, "failed to compute hash", status)
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(result.Hash)) // nolint:errcheck
		}
	}
}
