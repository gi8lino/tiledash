package render

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/hash"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/templates"
)

// Result captures the rendered HTML and hash for a tile.
type Result struct {
	HTML string
	Hash string
}

// TileRenderer renders tiles via runners, caches the rendered output per tile,
// and exposes both the HTML and its hash (computed from the rendered HTML).
type TileRenderer struct {
	cfg      config.DashboardConfig
	runners  []providers.Runner
	tileTmpl *template.Template
	logger   *slog.Logger

	mu    sync.RWMutex
	cache []cachedTile
}

type cachedTile struct {
	rendered Result
	expires  time.Time
}

// NewTileRenderer constructs a renderer over the provided runners and template set.
func NewTileRenderer(cfg config.DashboardConfig, runners []providers.Runner, tmpl *template.Template, logger *slog.Logger) *TileRenderer {
	return &TileRenderer{
		cfg:      cfg,
		runners:  runners,
		tileTmpl: tmpl,
		logger:   logger,
		cache:    make([]cachedTile, len(runners)),
	}
}

// RenderTile returns the rendered HTML and its hash for the given tile index,
// using a cached value when valid. On failure, it returns a RenderError with
// an HTTP status for callers to map into responses.
func (t *TileRenderer) RenderTile(ctx context.Context, idx int) (Result, int, *templates.RenderError) {
	if idx < 0 {
		return Result{}, http.StatusBadRequest, templates.NewRenderError("render", "Invalid tile id", "index out of range")
	}
	if idx >= len(t.runners) {
		return Result{}, http.StatusNotFound, templates.NewRenderError("render", "Invalid tile id", "index out of range")
	}

	ttl := t.cfg.Tiles[idx].Request.TTL

	// Fast path: return cached render if still fresh.
	if ttl > 0 {
		now := time.Now()
		t.mu.RLock()
		entry := t.cache[idx]
		t.mu.RUnlock()
		if !entry.expires.IsZero() && now.Before(entry.expires) {
			return entry.rendered, http.StatusOK, nil
		}
	}

	acc, pages, status, err := t.runners[idx].Do(ctx)
	if err != nil {
		if status == 0 {
			status = http.StatusBadGateway
		}
		t.logger.Error("fetch error", "id", idx, "status", status, "pages", pages, "error", err.Error())
		return Result{}, status, templates.NewRenderError("upstream", "request failed", err.Error())
	}

	html, renderErr := templates.RenderCell(ctx, idx, t.cfg, t.tileTmpl, acc)
	if renderErr != nil {
		t.logger.Error("render tile error", "id", idx, "error", renderErr.Error())
		return Result{}, http.StatusInternalServerError, renderErr
	}

	hash, hashErr := hash.Any(string(html))
	if hashErr != nil {
		t.logger.Error("hash computation failed", "id", idx, "error", hashErr.Error())
		return Result{}, http.StatusInternalServerError, templates.NewRenderError("hash", "failed to hash rendered tile", hashErr.Error())
	}

	result := Result{
		HTML: string(html),
		Hash: hash,
	}

	if ttl > 0 {
		t.mu.Lock()
		t.cache[idx] = cachedTile{
			rendered: result,
			expires:  time.Now().Add(ttl),
		}
		t.mu.Unlock()
	}

	return result, http.StatusOK, nil
}
