package server

import (
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/handlers"
	"github.com/gi8lino/tiledash/internal/middleware"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/render"
)

// NewRouter creates and wires the HTTP mux with handlers and middleware; mounts under routerPrefix  if provided.
func NewRouter(
	webFS fs.FS,
	cellTmpl *template.Template,
	errTmpl *template.Template,
	cfg config.DashboardConfig,
	logger *slog.Logger,
	runners []providers.Runner,
	debug bool,
	version string,
	routePrefix string,
) http.Handler {
	// Inner mux registers canonical routes rooted at "/".
	root := http.NewServeMux()

	renderer := render.NewTileRenderer(cfg, runners, cellTmpl, logger)

	// Serve embedded static files at /static/*.
	staticContent, _ := fs.Sub(webFS, "web/static")       // sub-FS with static assets
	fileServer := http.FileServer(http.FS(staticContent)) // file server for static
	root.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Health checks.
	root.Handle("GET /healthz", handlers.Healthz())
	root.Handle("POST /healthz", handlers.Healthz())

	// Main dashboard handler.
	root.Handle("/", handlers.BaseHandler(webFS, routePrefix, version, cfg, renderer, logger))

	// API endpoints (tile content + tile hash), exposed under /api/v1/*.
	api := http.NewServeMux()
	api.Handle("GET /tile/{id}", handlers.TileHandler(renderer, errTmpl, logger))
	api.Handle("GET /hash/{id}", handlers.HashHandler(cfg, renderer, logger))
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
