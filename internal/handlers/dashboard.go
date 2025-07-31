package handlers

import (
	"io/fs"
	"net/http"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/templates"
)

// Dashboard returns the main dashboard HTTP handler.
func Dashboard(webFS fs.FS, templateDir string, version string, c *jira.Client, cfg config.DashboardConfig) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()

	baseTmpl := templates.ParseBaseTemplates(webFS, funcMap)
	sectionTmpl := templates.ParseSectionTemplates(templateDir, funcMap)

	return func(w http.ResponseWriter, r *http.Request) {
		sections, status, err := templates.RenderSections(r.Context(), cfg, sectionTmpl, c)
		if err != nil {
			http.Error(w, err.Error(), status)
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
			http.Error(w, "render base error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
