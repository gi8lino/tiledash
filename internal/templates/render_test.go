package templates

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"testing"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderError(t *testing.T) {
	t.Parallel()

	t.Run("Error message", func(t *testing.T) {
		t.Parallel()
		err := NewRenderError("test-type", "test-message", "test-detail")
		assert.Equal(t, err.Error(), "test-type: test-message (test-detail)")
	})
}

func TestRenderCell(t *testing.T) {
	t.Parallel()

	t.Run("renders valid cell", func(t *testing.T) {
		t.Parallel()

		cellTmpl := template.Must(template.New("").Parse(`{{define "s1"}}<div>{{.Title}}: {{.Data.key}}</div>{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"key": "value"}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{
				{
					Title:    "Test",
					Query:    "project = TEST",
					Template: "s1",
				},
			},
		}

		html, err := RenderCell(context.Background(), 0, cfg, cellTmpl, client)
		assert.Nil(t, err)
		assert.Contains(t, string(html), "Test: value")
	})

	t.Run("handles fetch error", func(t *testing.T) {
		t.Parallel()

		cellTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, http.StatusTeapot, errors.New("backend error")
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{{Title: "Fail", Query: "bad", Template: "x"}},
		}

		html, renderErr := RenderCell(context.Background(), 0, cfg, cellTmpl, client)
		require.Error(t, renderErr)
		assert.EqualError(t, renderErr, "fetch: Request failed: status 418 (backend error)")
		assert.Empty(t, html)
	})

	t.Run("handles JSON parsing error", func(t *testing.T) {
		t.Parallel()

		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{invalid json}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{{Title: "Broken JSON", Query: "test", Template: "s1"}},
		}

		html, renderErr := RenderCell(context.Background(), 0, cfg, errTmpl, client)
		require.Error(t, renderErr)
		assert.EqualError(t, renderErr, "json: Response could not be parsed (invalid character 'i' looking for beginning of object key string)")
		assert.Empty(t, html)
	})

	t.Run("handles template render error", func(t *testing.T) {
		t.Parallel()

		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"foo": "bar"}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{{Title: "Template Fail", Query: "test", Template: "s1"}},
		}

		html, renderErr := RenderCell(context.Background(), 0, cfg, errTmpl, client)
		require.Error(t, renderErr)
		assert.EqualError(t, renderErr, "template: Template rendering failed (html/template: \"s1\" is undefined)")
		assert.Empty(t, html)
	})

	t.Run("handles invalid index", func(t *testing.T) {
		t.Parallel()

		cellTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &jira.Client{}
		cfg := config.DashboardConfig{} // empty Cells

		html, renderErr := RenderCell(context.Background(), 42, cfg, cellTmpl, client)
		require.Error(t, renderErr)
		assert.EqualError(t, renderErr, "render: Failed to get section (cell index 42 out of bounds)")
		assert.Empty(t, html)
	})
}
