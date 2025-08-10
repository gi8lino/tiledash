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

// NewRouter creates and wires the HTTP mux with handlers and middleware.
func NewRouter(
	webFS fs.FS,
	templateDir string,
	cfg config.DashboardConfig,
	logger *slog.Logger,
	runners []providers.Runner, // interface slice (no pointer-to-interface)
	debug bool,
	version string,
) http.Handler {
	root := http.NewServeMux()

	// Serve embedded static files.
	staticContent, _ := fs.Sub(webFS, "web/static")
	fileServer := http.FileServer(http.FS(staticContent))
	root.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Health checks.
	root.Handle("GET /healthz", handlers.Healthz())
	root.Handle("POST /healthz", handlers.Healthz())

	// Main dashboard handler.
	root.Handle("/", handlers.BaseHandler(webFS, templateDir, version, cfg, logger))

	// API endpoints (tile content + tile hash).
	api := http.NewServeMux()
	api.Handle("GET /tile/{id}", handlers.TileHandler(webFS, templateDir, version, cfg, runners, logger))
	api.Handle("GET /hash/{id}", handlers.HashHandler(cfg, logger))

	// Mount API under /api/v1/.
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", api))

	if debug {
		// Attach request logging when debug is enabled.
		return middleware.Chain(root, middleware.LoggingMiddleware(logger))
	}
	return root
}
