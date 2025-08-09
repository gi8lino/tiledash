package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/templates"
)

// BaseHandler returns a handler function that renders a dashboard.
func BaseHandler(
	webFS fs.FS,
	templateDir string,
	version string,
	s jira.Searcher,
	cfg config.DashboardConfig,
	logger *slog.Logger,
) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()
	baseTmpl := templates.ParseBaseTemplates(webFS, funcMap)

	return func(w http.ResponseWriter, r *http.Request) {
		cfgHash, _ := hashAny(cfg)
		computeCellHashes(&cfg, logger) // Compute hashes for all cells

		if err := baseTmpl.ExecuteTemplate(w, "base", map[string]any{
			"Version":         version,
			"Grid":            cfg.Grid,
			"Title":           cfg.Title,
			"RefreshInterval": int(cfg.RefreshInterval.Seconds()),
			"Customization":   &cfg.Customization,
			"Cells":           cfg.Cells, // pass cells directly for async placeholder generation
			"ConfigHash":      cfgHash,
		}); err != nil {
			renderErrorPage(w, http.StatusInternalServerError, baseTmpl, "Error", "Failed to render dashboard cells.", err)
		}
	}
}

// computeCellHashes hashes each cell in the given config.
func computeCellHashes(cfg *config.DashboardConfig, logger *slog.Logger) {
	for i := range cfg.Cells {
		cell := &cfg.Cells[i]
		h, err := hashAny(cell)
		if err != nil {
			logger.Error("hash computation failed", "id", i, "error", err)
			continue
		}
		cell.Hash = h // Add this field to config.Cell
	}
}
