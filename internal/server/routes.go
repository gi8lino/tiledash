package server

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/handlers"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/middleware"
)

// NewRouter creates a new HTTP router.
func NewRouter(
	webFS fs.FS,
	templateDir string,
	client *jira.Client,
	cfg config.DashboardConfig,
	logger *slog.Logger,
	debug bool,
	version string,
) http.Handler {
	root := http.NewServeMux()

	// Serve embedded static files
	staticContent, _ := fs.Sub(webFS, "web/static")
	fileServer := http.FileServer(http.FS(staticContent))
	root.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Health checks (no logging)
	root.Handle("GET /healthz", handlers.Healthz())
	root.Handle("POST /healthz", handlers.Healthz())

	// Main dashboard handler (with optional logging)
	var dashboardHandler http.Handler = handlers.Dashboard(webFS, templateDir, version, client, cfg)
	if debug {
		dashboardHandler = middleware.Chain(dashboardHandler, middleware.LoggingMiddleware(logger))
	}

	root.Handle("GET /", dashboardHandler)

	return root
}
