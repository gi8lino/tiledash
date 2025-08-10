package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gi8lino/tiledash/internal/templates"
)

// renderErrorPage renders a simple error page. Never panics; always writes something.
func renderErrorPage(
	w http.ResponseWriter,
	status int,
	pageErrTmpl *template.Template, // must contain "page_error"
	title, msg string,
	err error,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	data := struct {
		Title   string
		Message string
		Error   string
	}{
		Title:   title,
		Message: msg,
	}
	if err != nil {
		data.Error = err.Error()
	}

	if tplErr := pageErrTmpl.ExecuteTemplate(w, "page_error", data); tplErr != nil {
		fmt.Fprintf(w, `<div class="alert alert-danger">Failed to render error page: %s</div>`, tplErr) // nolint:errcheck
	}
}

// renderCellError renders a tile error using the "tile_error" template.
func renderCellError(
	w http.ResponseWriter,
	status int,
	tileErrTmpl *template.Template, // must contain "tile_error"
	re *templates.RenderError,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	if tplErr := tileErrTmpl.ExecuteTemplate(w, "tile_error", re); tplErr != nil {
		fmt.Fprintf(w, `<div class="alert alert-danger">Failed to render tile error: %s</div>`, tplErr)
	}
}
