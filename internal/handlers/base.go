package handlers

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/hash"
	"github.com/gi8lino/tiledash/internal/render"
	"github.com/gi8lino/tiledash/internal/templates"
)

// BaseHandler returns a handler function that renders a dashboard.
func BaseHandler(
	webFS fs.FS,
	routePrefix string,
	version string,
	cfg config.DashboardConfig,
	renderer *render.TileRenderer,
	logger *slog.Logger,
) http.HandlerFunc {
	funcMap := templates.TemplateFuncMap()
	baseTmpl := templates.ParseBaseTemplates(webFS, funcMap)

	return func(w http.ResponseWriter, r *http.Request) {
		cfgHash, _ := hash.Any(cfg)
		computeRenderedHashes(r.Context(), renderer, &cfg, logger)

		if err := baseTmpl.ExecuteTemplate(w, "base", map[string]any{
			"Version":         version,
			"Grid":            cfg.Grid,
			"Title":           cfg.Title,
			"RoutePrefix":     routePrefix,
			"RefreshInterval": int(cfg.RefreshInterval.Seconds()),
			"Customization":   &cfg.Customization,
			"Cells":           cfg.Tiles, // pass tiles directly for async placeholder generation
			"ConfigHash":      cfgHash,
		}); err != nil {
			renderErrorPage(w, http.StatusInternalServerError, baseTmpl, "Error", "Failed to render dashboard tiles.", err)
		}
	}
}

// computeRenderedHashes populates cfg.Tiles[*].Hash using the rendered HTML of each tile.
func computeRenderedHashes(ctx context.Context, renderer *render.TileRenderer, cfg *config.DashboardConfig, logger *slog.Logger) {
	if renderer == nil {
		return
	}

	for i := range cfg.Tiles {
		result, _, err := renderer.RenderTile(ctx, i)
		if err != nil {
			logger.Error("hash computation failed", "id", i, "error", err)
			continue
		}
		cfg.Tiles[i].Hash = result.Hash
	}
}
