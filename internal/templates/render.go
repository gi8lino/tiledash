package templates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
)

// RenderSections fetches data and renders each dashboard section.
func RenderSections(ctx context.Context, cfg config.DashboardConfig, tmpl *template.Template, client *jira.Client) (sections []map[string]any, statusCode int, err error) {
	for _, section := range cfg.Layout {
		respBody, status, err := client.SearchByJQL(ctx, section.Query, section.Params)
		if err != nil {
			return nil, status, fmt.Errorf("fetch error: %w", err)
		}

		var jsonData any
		if err := json.Unmarshal(respBody, &jsonData); err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("json error: %w", err)
		}

		var buf bytes.Buffer
		err = tmpl.ExecuteTemplate(&buf, section.Template, map[string]any{
			"Title": section.Title,
			"Data":  jsonData,
		})
		if err != nil {
			return nil, http.StatusInternalServerError, fmt.Errorf("template error: %w", err)
		}

		sections = append(sections, map[string]any{
			"HTML":    template.HTML(buf.String()),
			"Row":     section.Position.Row,
			"Col":     section.Position.Col,
			"ColSpan": section.Position.ColSpan,
			"Title":   section.Title,
		})
	}

	return sections, http.StatusOK, nil
}
