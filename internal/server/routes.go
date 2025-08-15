package server

import (
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/handlers"
	"github.com/gi8lino/tiledash/internal/middleware"
	"github.com/gi8lino/tiledash/internal/providers"
)

// NewRouter creates and wires the HTTP mux with handlers and middleware; mounts under routerPrefix  if provided.
func NewRouter(
	webFS fs.FS,
	templateDir string,
	cfg config.DashboardConfig,
	logger *slog.Logger,
	runners []providers.Runner,
	debug bool,
	version string,
	routePrefix string,
) http.Handler {
	// Inner mux registers canonical routes rooted at "/".
	root := http.NewServeMux()

	// Serve embedded static files at /static/*.
	staticContent, _ := fs.Sub(webFS, "web/static")       // sub-FS with static assets
	fileServer := http.FileServer(http.FS(staticContent)) // file server for static
	root.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Health checks.
	root.Handle("GET /healthz", handlers.Healthz())
	root.Handle("POST /healthz", handlers.Healthz())

	// Main dashboard handler.
	root.Handle("/", handlers.BaseHandler(webFS, templateDir, routePrefix, version, cfg, logger))

	// API endpoints (tile content + tile hash), exposed under /api/v1/*.
	api := http.NewServeMux()
	api.Handle("GET /tile/{id}", handlers.TileHandler(webFS, templateDir, version, cfg, runners, logger))
	api.Handle("GET /hash/{id}", handlers.HashHandler(cfg, logger))
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", api))

	// Mount the whole app under the prefix if provided
	var handler http.Handler = root
	if routePrefix != "" {
		handler = mountUnderPrefix(root, routePrefix)
	}

	// Optional debug logging middleware.
	if debug {
		return middleware.Chain(handler, middleware.LoggingMiddleware(logger))
	}
	return handler
}
