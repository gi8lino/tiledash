package templates

import (
	"html/template"
	"io/fs"
	"path/filepath"
)

// ParseBaseTemplates loads the base, footer, and error layout templates from webFS.
func ParseBaseTemplates(webFS fs.FS, funcMap template.FuncMap) *template.Template {
	// Use New("jirapanel") to avoid assigning a name to the root template.
	// This ensures that files like base.gohtml must explicitly declare {{define "base"}}
	// in order to be invoked via ExecuteTemplate("base", ...), preventing accidental fallback behavior.
	return template.Must(
		template.New("jirapanel").
			Funcs(funcMap).
			ParseFS(webFS,
				"web/templates/base.gohtml",
				"web/templates/footer.gohtml",
				"web/templates/error.gohtml",
			),
	)
}

// ParseSectionTemplates parses user-defined section templates, ignoring missing files.
func ParseSectionTemplates(templateDir string, funcMap template.FuncMap) (*template.Template, error) {
	tmpl := template.New("").Funcs(funcMap)

	matches, err := filepath.Glob(filepath.Join(templateDir, "*.gohtml"))
	if err != nil {
		return nil, err // actual glob error
	}

	if len(matches) == 0 {
		return tmpl, nil // return empty template set without error
	}

	parsed, err := tmpl.ParseFiles(matches...)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}
