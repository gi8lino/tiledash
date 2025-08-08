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
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCellHandler(t *testing.T) {
	t.Parallel()

	t.Run("renders cell successfully", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/cell_error.gohtml": &fstest.MapFile{Data: []byte(`{{define "cell_error"}}ERROR: {{.Message}}{{end}}`)},
		}

		tmpDir := t.TempDir()
		testutils.MustWriteFile(t,
			filepath.Join(tmpDir, "cell.gohtml"),
			`{{define "cell.html"}}<div>Cell: {{.Title}}</div>{{end}}`,
		)

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Title:    "Test Cell",
					Query:    "project = TEST",
					Template: "cell.html",
				},
			},
			RefreshInterval: 5 * time.Second,
		}

		mockClient := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"issues":[]}`), http.StatusOK, nil
			},
		}

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/layout/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		handler := CellHandler(webFS, tmpDir, "vX", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, string(body), "Cell: Test Cell")
	})

	t.Run("renders error on JQL failure", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/cell_error.gohtml": &fstest.MapFile{Data: []byte(`{{define "cell_error"}}Error: {{.Message}}{{end}}`)},
		}

		tmpDir := t.TempDir()
		testutils.MustWriteFile(t,
			filepath.Join(tmpDir, "dummy.gohtml"),
			`{{define "dummy"}}noop{{end}}`,
		)

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Title:    "Broken Cell",
					Query:    "invalid-jql",
					Template: "dummy",
				},
			},
		}

		mockClient := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, http.StatusInternalServerError, assert.AnError
			},
		}

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/layout/0", nil)
		req.SetPathValue("id", "0")

		w := httptest.NewRecorder()

		handler := CellHandler(webFS, tmpDir, "dev", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)

		require.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Contains(t, string(body), "Error: Request failed")
	})

	t.Run("invalid id returns error", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/cell_error.gohtml": &fstest.MapFile{Data: []byte(`{{define "cell_error"}}invalid{{end}}`)},
		}

		cfg := config.DashboardConfig{}
		mockClient := &jira.Client{}

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/layout/foo", nil)
		req.SetPathValue("id", "foo")

		w := httptest.NewRecorder()

		handler := CellHandler(webFS, ".", "x", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})

	t.Run("missing id returns error", func(t *testing.T) {
		t.Parallel()

		webFS := fstest.MapFS{
			"web/templates/cell_error.gohtml": &fstest.MapFile{Data: []byte(`{{define "cell_error"}}ERROR: {{.Message}}{{end}}`)},
		}
		cfg := config.DashboardConfig{}
		mockClient := &jira.Client{}

		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, nil))

		req := httptest.NewRequest(http.MethodGet, "/cell/", nil)
		// note: no SetPathValue

		w := httptest.NewRecorder()

		handler := CellHandler(webFS, ".", "x", mockClient, cfg, logger)
		handler.ServeHTTP(w, req)

		res := w.Result()
		defer res.Body.Close() // nolint:errcheck

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
	})
}
