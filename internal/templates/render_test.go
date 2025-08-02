package templates_test

import (
	"context"
	"errors"
	"html/template"
	"net/http"
	"testing"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/templates"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockClient implements the jira.Client interface with configurable behavior.
type mockClient struct {
	searchFn func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error)
}

func (m *mockClient) SearchByJQL(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
	return m.searchFn(ctx, jql, params)
}

func TestRenderSections(t *testing.T) {
	t.Parallel()

	t.Run("renders valid section", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}<div>{{.Title}}: {{.Data.key}}</div>{{end}}`))
		client := &mockClient{
			searchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"key":"value"}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Layout: []config.Section{
				{
					Title:    "Test",
					Query:    "project = TEST",
					Template: "s1",
					Position: config.Position{Row: 0, Col: 1, ColSpan: 2},
				},
			},
		}

		sections, status, err := templates.RenderSections(context.Background(), cfg, tmpl, client)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		require.Len(t, sections, 1)
		assert.Equal(t, "Test", sections[0]["Title"])
		assert.Equal(t, 0, sections[0]["Row"])
		assert.Equal(t, 1, sections[0]["Col"])
		assert.Equal(t, 2, sections[0]["ColSpan"])
		assert.Contains(t, sections[0]["HTML"], "Test: value")
	})

	t.Run("handles fetch error", func(t *testing.T) {
		t.Parallel()

		tmpl := template.New("")
		client := &mockClient{
			searchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, 418, errors.New("bad request")
			},
		}

		cfg := config.DashboardConfig{
			Layout: []config.Section{{Title: "Err", Query: "fail", Template: "x"}},
		}

		sections, status, err := templates.RenderSections(context.Background(), cfg, tmpl, client)
		require.Error(t, err)
		assert.Nil(t, sections)
		assert.Equal(t, 418, status)
		assert.Contains(t, err.Error(), "fetch error")
	})

	t.Run("handles invalid JSON", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}hi{{end}}`))
		client := &mockClient{
			searchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{invalid json}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Layout: []config.Section{{Title: "bad json", Query: "q", Template: "s1"}},
		}

		sections, status, err := templates.RenderSections(context.Background(), cfg, tmpl, client)
		require.Error(t, err)
		assert.Nil(t, sections)
		assert.Equal(t, http.StatusInternalServerError, status)
		assert.Contains(t, err.Error(), "json error")
	})

	t.Run("handles template error", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Must(template.New("").Parse(`{{define "s1"}}{{index nil 0}}{{end}}`))
		client := &mockClient{
			searchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return []byte(`{"foo":"bar"}`), http.StatusOK, nil
			},
		}

		cfg := config.DashboardConfig{
			Layout: []config.Section{{Title: "bad tmpl", Query: "q", Template: "s1"}},
		}

		sections, status, err := templates.RenderSections(context.Background(), cfg, tmpl, client)
		require.Error(t, err)
		assert.Nil(t, sections)
		assert.Equal(t, http.StatusInternalServerError, status)
		assert.Contains(t, err.Error(), "template error")
	})
}
