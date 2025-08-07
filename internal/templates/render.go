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
		err := &RenderError{
			Type:    "render",
			Message: "Failed to get section",
			Detail:  err.Error(),
		}
		return renderErrorHTML(errTmpl, err), err

	}

	respBody, status, fetchErr := client.SearchByJQL(ctx, cell.Query, cell.Params)
	if fetchErr != nil {
		err := &RenderError{
			Type:    "fetch",
			Message: fmt.Sprintf("Request failed: status %d", status),
			Detail:  fetchErr.Error(),
		}
		return renderErrorHTML(errTmpl, err), err
	}

	var jsonData any
	if err := json.Unmarshal(respBody, &jsonData); err != nil {
		err := &RenderError{
			Type:    "json",
			Message: "Response could not be parsed",
			Detail:  err.Error(),
		}
		return renderErrorHTML(errTmpl, err), err
	}

	var buf bytes.Buffer
	if err := sectionTmpl.ExecuteTemplate(&buf, cell.Template, map[string]any{
		"ID":    id,
		"Title": cell.Title,
		"Data":  jsonData,
	}); err != nil {
		err := &RenderError{
			Type:    "template",
			Message: "Template rendering failed",
			Detail:  err.Error(),
		}
		return renderErrorHTML(errTmpl, err), err
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
