package templates

import (
	"html/template"
	"io/fs"
	"path"
)

// ParseBaseTemplates parses the base and footer templates.
func ParseBaseTemplates(webFS fs.FS, funcMap template.FuncMap) *template.Template {
	return template.Must(
		template.New("base").
			Funcs(funcMap).
			ParseFS(webFS,
				path.Join("web/templates", "base.gohtml"),
				path.Join("web/templates", "footer.gohtml"),
				path.Join("web/templates", "error.gohtml"),
			),
	)
}

// ParseSectionTemplates parses the user-defined section templates.
func ParseSectionTemplates(fsys fs.FS, funcMap template.FuncMap) *template.Template {
	tmpl := template.New("").Funcs(funcMap)

	parsed, err := tmpl.ParseFS(fsys, "web/templates/*.gohtml")
	if err != nil {
		panic(err) // keep Must-like behavior
	}
	return parsed
}
