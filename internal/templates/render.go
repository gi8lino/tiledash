package templates

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/jira"
)

// RenderSections fetches data and renders each dashboard section.
// It always returns a slice (possibly containing sections with errors).
func RenderSections(ctx context.Context, cfg config.DashboardConfig, tmpl *template.Template, client jira.Searcher) (sections []map[string]any) {
	for _, section := range cfg.Layout {
		respBody, status, fetchErr := client.SearchByJQL(ctx, section.Query, section.Params)
		if fetchErr != nil {
			// Render error instead of HTML
			sections = append(sections, map[string]any{
				"HTML": template.HTML(""), // no HTML content
				"Error": map[string]string{
					"Type":    "fetch",
					"Message": fmt.Sprintf("Request failed: status %d", status),
					"Detail":  fetchErr.Error(),
				},
				"Row":     section.Position.Row,
				"Col":     section.Position.Col,
				"ColSpan": section.Position.ColSpan,
				"Title":   section.Title,
			})
			continue
		}

		var jsonData any
		if err := json.Unmarshal(respBody, &jsonData); err != nil {
			sections = append(sections, map[string]any{
				"HTML": template.HTML(""),
				"Error": map[string]string{
					"Type":    "json",
					"Message": "Response could not be parsed",
					"Detail":  err.Error(),
				},
				"Row":     section.Position.Row,
				"Col":     section.Position.Col,
				"ColSpan": section.Position.ColSpan,
				"Title":   section.Title,
			})
			continue
		}

		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, section.Template, map[string]any{
			"Title": section.Title,
			"Data":  jsonData,
		}); err != nil {
			sections = append(sections, map[string]any{
				"HTML": template.HTML(""),
				"Error": map[string]string{
					"Type":    "template",
					"Message": "Template rendering failed",
					"Detail":  err.Error(),
				},
				"Row":     section.Position.Row,
				"Col":     section.Position.Col,
				"ColSpan": section.Position.ColSpan,
				"Title":   section.Title,
			})
			continue
		}

		sections = append(sections, map[string]any{
			"HTML":    template.HTML(buf.String()),
			"Error":   nil,
			"Row":     section.Position.Row,
			"Col":     section.Position.Col,
			"ColSpan": section.Position.ColSpan,
			"Title":   section.Title,
		})
	}

	return sections
}
