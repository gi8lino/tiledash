package handlers

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboard(t *testing.T) {
	t.Parallel()

	t.Run("renders dashboard successfully", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}{{.Title}} v{{.Version}}{{end}}`)},
			"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}Error: {{.Message}}{{end}}`)},
		}

		// Create cell template with expected name
		tmpDir := t.TempDir()
		testutils.MustWriteFile(t, filepath.Join(tmpDir, "templates/cell_example.gohtml"), `{{define "cell_example.html"}}<div>{{.Title}}</div>{{end}}`)

		cfg := config.DashboardConfig{
			Title: "My Dashboard",
			Grid: config.Grid{
				Columns: 3,
				Rows:    2,
			},
			Cells: []config.Cell{
				{
					Title:    "Example Section",
					Query:    "project = TEST",
					Template: "cell_example.html",
					Position: config.Position{Row: 0, Col: 0},
				},
			},
			RefreshInterval: 30 * time.Second,
		}

		mockClient := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"issues": []}`), http.StatusOK, nil
			},
		}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		// Use tmpDir/templates as the cell template directory
		handler := BaseHandler(webFS, filepath.Join(tmpDir, "templates"), "1.0.0", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body := w.Body.String()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "My Dashboard v1.0.0")
	})

	t.Run("renders cell error in base template", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/base.gohtml": &fstest.MapFile{Data: []byte(`
{{define "base"}}
<div class="grid">
  {{range .Cells}}
    <div class="card">
      {{if .Error}}
        <div class="alert alert-danger">
          {{.Error.Type}}: {{.Error.Message}} ({{.Error.Detail}})
        </div>
      {{end}}
    </div>
  {{end}}
</div>
{{end}}
`)},
			"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}Error: {{.Message}}{{end}}`)},
		}

		tmpDir := t.TempDir()
		testutils.MustWriteFile(t,
			filepath.Join(tmpDir, "templates", "dummy.gohtml"),
			`{{define "dummy"}}noop{{end}}`,
		)

		cfg := config.DashboardConfig{
			Title: "Broken Dashboard",
			Cells: []config.Cell{
				{
					Title:    "Failing Section",
					Query:    "FAIL-JQL",
					Template: "dummy",
				},
			},
			RefreshInterval: 10 * time.Second,
		}

		mockClient := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, http.StatusInternalServerError, assert.AnError
			},
		}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := BaseHandler(webFS, filepath.Join(tmpDir, "templates"), "dev", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(body), "Failed to render dashboard cells")
	})
}
