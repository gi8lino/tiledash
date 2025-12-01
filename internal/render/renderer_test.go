package render

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/hash"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/templates"
)

type countingRunner struct {
	count  *int32
	acc    providers.Accumulator
	status int
	err    error
}

func (f countingRunner) Do(ctx context.Context) (providers.Accumulator, int, int, error) {
	atomic.AddInt32(f.count, 1)
	return f.acc, 1, f.status, f.err
}

func TestRenderTileSuccess(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}<div>{{index .Data "v"}}</div>{{end}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := config.DashboardConfig{
		Tiles: []config.Tile{{
			Title:    "Example",
			Template: "tile.gohtml",
			Request:  config.Request{},
		}},
	}

	funcMap := templates.TemplateFuncMap()
	tmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	acc := providers.Accumulator{"merged": map[string]any{"v": "ok"}}
	count := int32(0)
	runners := []providers.Runner{
		countingRunner{count: &count, acc: acc, status: http.StatusOK},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	renderer := NewTileRenderer(cfg, runners, tmpl, logger)

	result, status, renderErr := renderer.RenderTile(context.Background(), 0)
	if renderErr != nil {
		t.Fatalf("render error: %v", renderErr)
	}
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	expectedHash, err := hash.Any(result.HTML)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if result.Hash != expectedHash {
		t.Fatalf("unexpected hash: %q != %q", result.Hash, expectedHash)
	}
	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected runner called once, got %d", count)
	}
}

func TestRenderTileCachesWithinTTL(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}cached{{end}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := config.DashboardConfig{
		Tiles: []config.Tile{{
			Title:    "Cached",
			Template: "tile.gohtml",
			Request:  config.Request{TTL: time.Hour},
		}},
	}
	funcMap := templates.TemplateFuncMap()
	tmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	count := int32(0)
	runners := []providers.Runner{
		countingRunner{count: &count, acc: providers.Accumulator{}, status: http.StatusOK},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	renderer := NewTileRenderer(cfg, runners, tmpl, logger)

	if _, _, err := renderer.RenderTile(context.Background(), 0); err != nil {
		t.Fatalf("first render error: %v", err)
	}
	if _, _, err := renderer.RenderTile(context.Background(), 0); err != nil {
		t.Fatalf("second render error: %v", err)
	}

	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected single runner call due to caching, got %d", count)
	}
}

func TestRenderTileCachesEmptyHTML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}{{end}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := config.DashboardConfig{
		Tiles: []config.Tile{{
			Title:    "Empty",
			Template: "tile.gohtml",
			Request:  config.Request{TTL: time.Minute},
		}},
	}
	funcMap := templates.TemplateFuncMap()
	tmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	count := int32(0)
	runners := []providers.Runner{
		countingRunner{count: &count, acc: providers.Accumulator{}, status: http.StatusOK},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	renderer := NewTileRenderer(cfg, runners, tmpl, logger)

	if _, _, err := renderer.RenderTile(context.Background(), 0); err != nil {
		t.Fatalf("first render error: %v", err)
	}
	if _, _, err := renderer.RenderTile(context.Background(), 0); err != nil {
		t.Fatalf("second render error: %v", err)
	}

	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected empty HTML to be cached, got runner calls: %d", count)
	}
}

func TestRenderTileExpiresAfterTTL(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}expire{{end}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := config.DashboardConfig{
		Tiles: []config.Tile{{
			Title:    "Expire",
			Template: "tile.gohtml",
			Request:  config.Request{TTL: 5 * time.Millisecond},
		}},
	}
	funcMap := templates.TemplateFuncMap()
	tmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	count := int32(0)
	runners := []providers.Runner{
		countingRunner{count: &count, acc: providers.Accumulator{}, status: http.StatusOK},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	renderer := NewTileRenderer(cfg, runners, tmpl, logger)

	if _, _, err := renderer.RenderTile(context.Background(), 0); err != nil {
		t.Fatalf("first render error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if _, _, err := renderer.RenderTile(context.Background(), 0); err != nil {
		t.Fatalf("second render error: %v", err)
	}

	if atomic.LoadInt32(&count) != 2 {
		t.Fatalf("expected cache to expire and trigger second runner call, got %d", count)
	}
}

func TestRenderTileInvalidID(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}noop{{end}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := config.DashboardConfig{Tiles: []config.Tile{}}
	funcMap := templates.TemplateFuncMap()
	tmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	renderer := NewTileRenderer(cfg, []providers.Runner{}, tmpl, logger)

	if _, status, err := renderer.RenderTile(context.Background(), -1); err == nil || status != http.StatusBadRequest {
		t.Fatalf("expected bad request for negative id, got status %d, err %v", status, err)
	}
	if _, status, err := renderer.RenderTile(context.Background(), 5); err == nil || status != http.StatusNotFound {
		t.Fatalf("expected not found for out of range id, got status %d, err %v", status, err)
	}
}

func TestRenderTileUpstreamError(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "tile.gohtml"), []byte(`{{define "tile.gohtml"}}noop{{end}}`), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	cfg := config.DashboardConfig{
		Tiles: []config.Tile{{
			Title:    "UpstreamFail",
			Template: "tile.gohtml",
			Request:  config.Request{},
		}},
	}
	funcMap := templates.TemplateFuncMap()
	tmpl, err := templates.ParseCellTemplates(tmpDir, funcMap)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	runErr := assertError("boom")
	count := int32(0)
	runners := []providers.Runner{
		countingRunner{count: &count, acc: nil, status: http.StatusBadGateway, err: runErr},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	renderer := NewTileRenderer(cfg, runners, tmpl, logger)

	if _, status, err := renderer.RenderTile(context.Background(), 0); err == nil || status != http.StatusBadGateway {
		t.Fatalf("expected upstream failure, got status %d, err %v", status, err)
	}
}

type assertError string

func (a assertError) Error() string { return string(a) }
