package handlers

import (
	"html/template"
	"net/http"
)

// renderErrorPage renders a simple error card. Never panics; always writes something.
func renderErrorPage(
	w http.ResponseWriter,
	status int,
	tmpl *template.Template, // should contain "cell_error"
	title, msg string,
	cause error,
) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Write the status before attempting template execution.
	w.WriteHeader(status)

	data := struct {
		Title   string
		Message string
		Error   string
	}{
		Title:   title,
		Message: msg,
		Error:   "",
	}
	if cause != nil {
		data.Error = cause.Error()
	}

	if tplErr := tmpl.ExecuteTemplate(w, "cell_error", data); tplErr != nil {
		w.Write([]byte("<div class=\"alert alert-danger\">Failed to render error page</div>")) // nolint:errcheck
	}
}
