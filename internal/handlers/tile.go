package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/templates"
)

// TileHandler serves a tile by index using precompiled runners.
func TileHandler(
	webFS fs.FS,
	templateDir string,
	version string,
	cfg config.DashboardConfig,
	runners []providers.Runner, // interface slice (not []*Runner)
	logger *slog.Logger,
) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()

	errTmpl := templates.ParseCellErrorTemplate(webFS, funcMap)
	tileTmpl, err := templates.ParseCellTemplates(templateDir, funcMap)
	if err != nil {
		panic(err) // fail early if templates are broken
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing tile id", http.StatusBadRequest)
			return
		}
		idx, convErr := strconv.Atoi(id)
		if convErr != nil || idx < 0 || idx >= len(runners) {
			logger.Error("invalid tile id", "id", id)
			renderCellError(w, http.StatusBadRequest, errTmpl,
				templates.NewRenderError("render", "Invalid tile id", "index out of range"))
			return
		}

		// Execute precompiled request; paginator (if any) is transparent here.
		acc, pages, status, err := runners[idx].Do(r.Context())
		if err != nil {
			logger.Error("fetch error", "status", status, "pages", pages, "error", err.Error())
			renderCellError(w, http.StatusBadGateway, errTmpl,
				templates.NewRenderError("upstream", "request failed", err.Error()))
			return
		}

		// Render template from accumulator. normalizeData handles merged/first page.
		html, renderErr := templates.RenderCell(r.Context(), idx, cfg, tileTmpl, acc)
		if renderErr != nil {
			logger.Error("render tile error", "error", renderErr.Error())
			renderCellError(w, http.StatusInternalServerError, errTmpl, renderErr)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}
}
