package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/templates"
)

// Dashboard returns the main dashboard HTTP handler.
func Dashboard(webFS fs.FS, templateDir string, version string, c *jira.Client, cfg config.DashboardConfig, logger *slog.Logger) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()

	baseTmpl := templates.ParseBaseTemplates(webFS, funcMap)
	sectionTmpl := templates.ParseSectionTemplates(templateDir, funcMap)

	return func(w http.ResponseWriter, r *http.Request) {
		sections, status, err := templates.RenderSections(r.Context(), cfg, sectionTmpl, c)
		if err != nil {
			logger.Error("render sections error", "error", err)
			renderErrorPage(w, status, baseTmpl, cfg.Title, "Failed to render dashboard sections.", err)
			return
		}

		err = baseTmpl.ExecuteTemplate(w, "base.html", map[string]any{
			"Version":         version,
			"Grid":            cfg.Grid,
			"Title":           cfg.Title,
			"Sections":        sections,
			"RefreshInterval": int(cfg.RefreshInterval.Seconds()), // pass as int
		})
		if err != nil {
			logger.Error("render base error", "error", err)
			renderErrorPage(w, status, baseTmpl, cfg.Title, "Failed to render dashboard layout.", err)
			return
		}
	}
}
