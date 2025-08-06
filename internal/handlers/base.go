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
		// Compute hashes for all cells
		computeHashes(&cfg, logger)
		cfgHash, _ := hashAny(cfg)
		err := baseTmpl.ExecuteTemplate(w, "base", map[string]any{
			"Version":         version,
			"Grid":            cfg.Grid,
			"Title":           cfg.Title,
			"RefreshInterval": int(cfg.RefreshInterval.Seconds()),
			"Customization":   &cfg.Customization,
			"Cells":           cfg.Cells, // pass cells directly for async placeholder generation
			"ConfigHash":      cfgHash,
		})
		if err != nil {
			logger.Error("render base error", "error", err)
			renderErrorPage(w, http.StatusInternalServerError, baseTmpl, cfg.Title, "Failed to render dashboard cells.", err)
			return
		}
	}
}

func computeHashes(cfg *config.DashboardConfig, logger *slog.Logger) {
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
