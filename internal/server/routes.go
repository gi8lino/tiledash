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
	client jira.Searcher,
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
	var dashboardHandler http.Handler = handlers.BaseHandler(webFS, templateDir, version, client, cfg, logger)
	if debug {
		dashboardHandler = middleware.Chain(dashboardHandler, middleware.LoggingMiddleware(logger))
	}
	root.Handle("/", dashboardHandler) // no Method allowed, otherwise it crashes

	// API endpoint for loading sections asynchronously
	api := http.NewServeMux()
	api.Handle("GET /cell/{id}", handlers.CellHandler(webFS, templateDir, version, client, cfg, logger))
	api.Handle("GET /hash/{id}", handlers.HashHandler(cfg, logger))

	// mount under /api/v1/
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", api))

	return root
}
