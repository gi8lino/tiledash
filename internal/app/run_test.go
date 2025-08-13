package app_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gi8lino/tiledash/internal/app"
	"github.com/gi8lino/tiledash/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	// Minimal embedded FS with required base/error templates.
	webFS := fstest.MapFS{
		"web/templates/base.gohtml":        &fstest.MapFile{Data: []byte(`{{define "base"}}ok{{end}}`)},
		"web/templates/css/page.gohtml":    &fstest.MapFile{Data: []byte(`{{define "css_page"}}css{{end}}`)},
		"web/templates/css/debug.gohtml":   &fstest.MapFile{Data: []byte(`{{define "css_debug"}}cssd{{end}}`)},
		"web/templates/footer.gohtml":      &fstest.MapFile{Data: []byte(`{{define "footer"}}f{{end}}`)},
		"web/templates/errors/page.gohtml": &fstest.MapFile{Data: []byte(`{{define "page_error"}}err{{end}}`)},
		"web/templates/errors/tile.gohtml": &fstest.MapFile{Data: []byte(`{{define "tile_error"}}terr{{end}}`)},
	}

	dummyEnv := func(string) string { return "" }

	t.Run("Success (minimal config, empty tiles, ephemeral port)", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()

		tmp := t.TempDir()
		cfgPath := filepath.Join(tmp, "config.yaml")
		tplDir := filepath.Join(tmp, "templates")

		// Minimal valid config; no providers/tiles needed for startup.
		testutils.MustWriteFile(t, cfgPath, `
title: My Dashboard
grid: { columns: 1, rows: 1 }
refreshInterval: 1s
`)
		// Ensure tile template dir exists with at least one valid .gohtml,
		// because handlers.TileHandler parses directory at router construction.
		testutils.MustWriteFile(t, filepath.Join(tplDir, "dummy.gohtml"), `{{define "dummy"}}dummy{{end}}`)

		args := []string{
			"--config=" + cfgPath,
			"--template-dir=" + tplDir,
			"--listen-address=127.0.0.1:0", // avoid port conflicts
		}

		var out bytes.Buffer
		err := app.Run(ctx, webFS, "v1", "deadbeef", args, &out, dummyEnv)
		require.NoError(t, err)
	})

	t.Run("Help requested prints usage and returns nil", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		var out bytes.Buffer
		err := app.Run(ctx, webFS, "v1.2.3", "abc", []string{"--help"}, &out, dummyEnv)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "Usage")
	})

	t.Run("Version requested prints version and returns nil", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		var out bytes.Buffer
		err := app.Run(ctx, webFS, "v9.8.7", "cafebabe", []string{"--version"}, &out, dummyEnv)
		require.NoError(t, err)
		assert.Contains(t, out.String(), "v9.8.7")
	})

	t.Run("Unknown flag surfaces parsing error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		var out bytes.Buffer
		err := app.Run(ctx, webFS, "vX", "yyy", []string{"--totally-unknown"}, &out, dummyEnv)
		require.Error(t, err)
		assert.EqualError(t, err, "parsing error: unknown flag: --totally-unknown")
	})

	t.Run("Missing config file surfaces load error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()

		var out bytes.Buffer
		// Intentionally point to a non-existent file
		args := []string{
			"--config=/nope/does-not-exist.yaml",
			"--template-dir=" + t.TempDir(),
			"--listen-address=127.0.0.1:0",
		}
		err := app.Run(ctx, webFS, "v1", "deadbeef", args, &out, dummyEnv)
		require.Error(t, err)
		assert.EqualError(t, err, "loading config error: read config: open /nope/does-not-exist.yaml: no such file or directory")
	})
}
