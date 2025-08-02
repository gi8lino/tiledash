package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/templates"
)

func Dashboard(
	webFS fs.FS,
	templateDir string, // can be kept or removed — unused now
	version string,
	c *jira.Client,
	cfg config.DashboardConfig,
	logger *slog.Logger,
) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()

	baseTmpl := templates.ParseBaseTemplates(webFS, funcMap)
	sectionTmpl := templates.ParseSectionTemplates(webFS, funcMap)

	return func(w http.ResponseWriter, r *http.Request) {
		sections, status, err := templates.RenderSections(r.Context(), cfg, sectionTmpl, c)
		if err != nil {
			logger.Error("render sections error", "error", err)
			renderErrorPage(w, status, baseTmpl, cfg.Title, "Failed to render dashboard sections.", err)
			return
		}

		err = baseTmpl.ExecuteTemplate(w, "base", map[string]any{
			"Version":         version,
			"Grid":            cfg.Grid,
			"Title":           cfg.Title,
			"Sections":        sections,
			"RefreshInterval": int(cfg.RefreshInterval.Seconds()),
		})
		if err != nil {
			logger.Error("render base error", "error", err)
			renderErrorPage(w, status, baseTmpl, cfg.Title, "Failed to render dashboard layout.", err)
			return
		}
	}
}
