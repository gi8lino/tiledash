package handlers

import (
	"bytes"
	"context"
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
)

type mockClient struct {
	searchFn func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error)
}

func (m *mockClient) SearchByJQL(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
	return m.searchFn(ctx, jql, params)
}

func TestDashboard(t *testing.T) {
	t.Parallel()

	t.Run("renders dashboard successfully", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}{{.Title}} v{{.Version}}{{end}}`)},
			"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}Error: {{.Message}}{{end}}`)},
		}

		// Create section template with expected name
		tmpDir := t.TempDir()
		testutils.MustWriteFile(t, filepath.Join(tmpDir, "templates/section_example.gohtml"), `{{define "section_example.html"}}<div>{{.Title}}</div>{{end}}`)

		cfg := config.DashboardConfig{
			Title: "My Dashboard",
			Grid: config.Grid{
				Columns: 3,
				Rows:    2,
			},
			Layout: []config.Section{
				{
					Title:    "Example Section",
					Query:    "project = TEST",
					Template: "section_example.html",
					Position: config.Position{Row: 0, Col: 0},
				},
			},
			RefreshInterval: 30 * time.Second,
		}

		mockClient := &mockClient{
			searchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"issues": []}`), http.StatusOK, nil
			},
		}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		// Use tmpDir/templates as the section template directory
		handler := Dashboard(webFS, filepath.Join(tmpDir, "templates"), "1.0.0", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body := w.Body.String()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "My Dashboard v1.0.0")
	})

	t.Run("renders section error in base template", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/base.gohtml": &fstest.MapFile{Data: []byte(`
{{define "base"}}
<div class="grid">
  {{range .Sections}}
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
			Layout: []config.Section{
				{
					Title:    "Failing Section",
					Query:    "FAIL-JQL",
					Template: "dummy",
				},
			},
			RefreshInterval: 10 * time.Second,
		}

		mockClient := &mockClient{
			searchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, http.StatusInternalServerError, assert.AnError
			},
		}

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler := Dashboard(webFS, filepath.Join(tmpDir, "templates"), "dev", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body := w.Body.String()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "fetch: Request failed: status 500")
		assert.Contains(t, body, "alert alert-danger")
	})

	t.Run("panic when template parsing fails", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}base{{end}}`)},
			"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}Error: {{.Message}}{{end}}`)},
		}
		cfg := config.DashboardConfig{}

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic but none occurred")
			}
		}()

		tmpDir := t.TempDir()
		// Write malformed template
		testutils.MustWriteFile(t, filepath.Join(tmpDir, "bad.gohtml"), `{{define "bad"}}{{end}`) // unclosed

		_ = Dashboard(webFS, tmpDir, "dev", nil, cfg, nil)
	})
}
