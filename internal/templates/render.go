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
	Type    string
	Message string
	Detail  string
}

// NewRenderError creates a new RenderError.
func NewRenderError(typ, msg string, detail any) *RenderError {
	return &RenderError{
		Type:    typ,
		Message: msg,
		Detail:  fmt.Sprint(detail),
	}
}

// Error implements the error interface.
func (e *RenderError) Error() string {
	return fmt.Sprintf("%s: %s (%s)", e.Type, e.Message, e.Detail)
}

// RenderCell fetches and renders a single dashboard layout by index.
func RenderCell(
	ctx context.Context,
	id int,
	cfg config.DashboardConfig,
	sectionTmpl *template.Template,
	errTmpl *template.Template,
	client jira.Searcher,
) (template.HTML, *RenderError) {
	cell, err := cfg.GetLayoutByIndex(id)
	if err != nil {
		re := NewRenderError("render", "Failed to get section", err.Error())
		return renderErrorHTML(errTmpl, re), re
	}

	respBody, status, fetchErr := client.SearchByJQL(ctx, cell.Query, cell.Params)
	if fetchErr != nil {
		re := NewRenderError("fetch", fmt.Sprintf("Request failed: status %d", status), fetchErr.Error())
		return renderErrorHTML(errTmpl, re), re
	}

	var jsonData any
	if err := json.Unmarshal(respBody, &jsonData); err != nil {
		re := NewRenderError("json", "Response could not be parsed", err.Error())
		return renderErrorHTML(errTmpl, re), re
	}

	var buf bytes.Buffer
	if err := sectionTmpl.ExecuteTemplate(&buf, cell.Template, map[string]any{
		"ID":    id,
		"Title": cell.Title,
		"Data":  jsonData,
	}); err != nil {
		re := NewRenderError("template", "Template rendering failed", err.Error())
		return renderErrorHTML(errTmpl, re), re
	}

	return template.HTML(buf.String()), nil
}

// renderErrorHTML renders a cell error using the "cell_error" template.
func renderErrorHTML(tmpl *template.Template, err *RenderError) template.HTML {
	var buf bytes.Buffer
	if e := tmpl.ExecuteTemplate(&buf, "cell_error", err); e != nil {
		panic(fmt.Errorf("failed to render cell_error template: %w", e))
	}
	return template.HTML(buf.String())
}
