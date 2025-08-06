package handlers

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/templates"
)

// CellHandler returns HTML for a single layout (by index).
func CellHandler(
	webFS fs.FS,
	templateDir string,
	version string,
	s jira.Searcher,
	cfg config.DashboardConfig,
	logger *slog.Logger,
) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()

	errTmpl := templates.ParseCellErrorTemplate(webFS, funcMap)
	cellTmpl, err := templates.ParseCellTemplates(templateDir, funcMap)
	if err != nil {
		panic(err) // fail early if templates are broken
	}

	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing cell id", http.StatusBadRequest)
			return
		}

		idx, err := strconv.Atoi(id)
		if err != nil {
			logger.Error("invalid cell id", "id", id)
			renderErrorPage(w, http.StatusBadRequest, cellTmpl, cfg.Title, "Invalid cell id", err)
		}

		html, renderErr := templates.RenderCell(r.Context(), idx, cfg, cellTmpl, errTmpl, s)
		if renderErr != nil {
			logger.Error("render cell error", "error", renderErr.Error())
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)

		// Always return HTML (either valid content or rendered error)
		w.Write([]byte(html)) // nolint:errcheck
	}
}
