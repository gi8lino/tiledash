package handlers

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/tiledash/internal/render"
	"github.com/gi8lino/tiledash/internal/templates"
)

// TileHandler serves a tile by index using precompiled runners and cached renders.
func TileHandler(renderer *render.TileRenderer, errTmpl *template.Template, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing tile id", http.StatusBadRequest)
			return
		}
		idx, convErr := strconv.Atoi(id)
		if convErr != nil {
			logger.Error("invalid tile id", "id", id)
			renderCellError(w, http.StatusBadRequest, errTmpl,
				templates.NewRenderError("render", "Invalid tile id", "index out of range"))
			return
		}

		result, status, renderErr := renderer.RenderTile(r.Context(), idx)
		if renderErr != nil {
			renderCellError(w, status, errTmpl, renderErr)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(result.HTML))
	}
}
