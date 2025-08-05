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

func TestRenderCell(t *testing.T) {
	t.Parallel()

	t.Run("renders valid cell", func(t *testing.T) {
		t.Parallel()

		sectionTmpl := template.Must(template.New("").Parse(`{{define "s1"}}<div>{{.Title}}: {{.Data.key}}</div>{{end}}`))
		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

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

		html, err := RenderCell(context.Background(), 0, cfg, sectionTmpl, errTmpl, client)
		assert.Nil(t, err)
		assert.Contains(t, string(html), "Test: value")
	})

	t.Run("handles fetch error", func(t *testing.T) {
		t.Parallel()

		sectionTmpl := template.New("")
		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, http.StatusTeapot, errors.New("backend error")
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{{Title: "Fail", Query: "bad", Template: "x"}},
		}

		html, renderErr := RenderCell(context.Background(), 0, cfg, sectionTmpl, errTmpl, client)
		require.NotNil(t, renderErr)
		assert.Equal(t, "fetch", renderErr.Type)
		assert.Contains(t, string(html), "ERROR: fetch - Request failed: status 418")
	})

	t.Run("handles JSON parsing error", func(t *testing.T) {
		t.Parallel()

		sectionTmpl := template.Must(template.New("").Parse(`{{define "s1"}}ok{{end}}`))
		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{invalid json}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{{Title: "Broken JSON", Query: "test", Template: "s1"}},
		}

		html, renderErr := RenderCell(context.Background(), 0, cfg, sectionTmpl, errTmpl, client)
		require.NotNil(t, renderErr)
		assert.Equal(t, "json", renderErr.Type)
		assert.Contains(t, renderErr.Detail, "invalid character")
		assert.Contains(t, string(html), "ERROR: json - Response could not be parsed")
	})

	t.Run("handles template render error", func(t *testing.T) {
		t.Parallel()

		sectionTmpl := template.Must(template.New("").Parse(`{{define "s1"}}{{index nil 0}}{{end}}`))
		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"foo": "bar"}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Cells: []config.Cell{{Title: "Template Fail", Query: "test", Template: "s1"}},
		}

		html, renderErr := RenderCell(context.Background(), 0, cfg, sectionTmpl, errTmpl, client)
		require.NotNil(t, renderErr)
		assert.Equal(t, "template", renderErr.Type)
		assert.Contains(t, renderErr.Detail, "index")
		assert.Contains(t, string(html), "ERROR: template - Template rendering failed")
	})

	t.Run("handles invalid index", func(t *testing.T) {
		t.Parallel()

		sectionTmpl := template.New("")
		errTmpl := template.Must(template.New("").Parse(`{{define "cell_error"}}ERROR: {{.Type}} - {{.Message}}{{end}}`))

		client := &jira.Client{}
		cfg := config.DashboardConfig{} // empty Cells

		html, renderErr := RenderCell(context.Background(), 42, cfg, sectionTmpl, errTmpl, client)
		require.NotNil(t, renderErr)
		assert.Equal(t, "render", renderErr.Type)
		assert.Contains(t, renderErr.Message, "Failed to get section")
		assert.Contains(t, string(html), "ERROR: render - Failed to get section")
	})
}

func TestRenderErrorHTML(t *testing.T) {
	t.Parallel()

	t.Run("renders valid error template", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("cell_error").Parse(`{{define "cell_error"}}Type: {{.Type}}, Msg: {{.Message}}, Detail: {{.Detail}}{{end}}`))

		err := &RenderError{
			Type:    "json",
			Message: "Failed to parse JSON",
			Detail:  "unexpected token",
		}

		html := renderErrorHTML(tmpl, err)
		assert.Contains(t, string(html), "Type: json")
		assert.Contains(t, string(html), "Msg: Failed to parse JSON")
		assert.Contains(t, string(html), "Detail: unexpected token")
	})

	t.Run("panics on missing template", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("other").Parse(`{{define "other"}}noop{{end}}`))

		err := &RenderError{
			Type:    "oops",
			Message: "bad",
			Detail:  "missing template",
		}

		assert.PanicsWithError(t,
			"failed to render cell_error template: html/template: \"cell_error\" is undefined",
			func() {
				renderErrorHTML(tmpl, err)
			},
		)
	})
}
