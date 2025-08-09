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

// RenderError is a generic error returned by RenderSection.
type RenderError struct {
	Title   string
	Message string
	Detail  string
}

// NewRenderError creates a new RenderError.
func NewRenderError(typ, msg string, detail any) *RenderError {
	return &RenderError{
		Title:   typ,
		Message: msg,
		Detail:  fmt.Sprint(detail),
	}
}

// Error implements the error interface.
func (e *RenderError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Title, e.Message, e.Detail)
}

// RenderCell fetches and renders a single dashboard layout by index.
func RenderCell(
	ctx context.Context,
	id int,
	cfg config.DashboardConfig,
	cellTmpl *template.Template,
	client jira.Searcher,
) (template.HTML, *RenderError) {
	cell, err := cfg.GetLayoutByIndex(id)
	if err != nil {
		return "", NewRenderError("render", "Failed to get section", err.Error())
	}

	respBody, status, fetchErr := client.SearchByJQL(ctx, cell.Query, cell.Params)
	if fetchErr != nil {
		return "", NewRenderError("fetch", fmt.Sprintf("Request failed: status %d", status), fetchErr.Error())
	}

	var jsonData any
	if err := json.Unmarshal(respBody, &jsonData); err != nil {
		return "", NewRenderError("json", "Response could not be parsed", err.Error())
	}

	var buf bytes.Buffer
	if err := cellTmpl.ExecuteTemplate(&buf, cell.Template, map[string]any{
		"ID":    id,
		"Title": cell.Title,
		"Data":  jsonData,
	}); err != nil {
		return "", NewRenderError("template", "Template rendering failed", err.Error())
	}

	return template.HTML(buf.String()), nil
}
