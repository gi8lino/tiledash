package handlers

import (
	"html/template"
	"net/http"
)

// renderErrorPage renders a fallback error page using the base template.
func renderErrorPage(w http.ResponseWriter, statusCode int, tmpl *template.Template, title, msg string, err error) {
	w.WriteHeader(statusCode)
	_ = tmpl.ExecuteTemplate(w, "error", map[string]any{
		"Title":   title,
		"Message": msg,
		"Error":   err.Error(),
	})
}
