package templates

import (
	"html/template"
	"io/fs"
	"path"
)

// ParseBaseTemplates parses the base and footer templates.
func ParseBaseTemplates(webFS fs.FS, funcMap template.FuncMap) *template.Template {
	return template.Must(
		template.New("base.html").
			Funcs(funcMap).
			ParseFS(webFS,
				path.Join("web/templates", "base.gohtml"),
				path.Join("web/templates", "footer.gohtml"),
			),
	)
}

// ParseSectionTemplates parses the user-defined section templates.
func ParseSectionTemplates(templateDir string, funcMap template.FuncMap) *template.Template {
	return template.Must(
		template.New("").
			Funcs(funcMap).
			ParseGlob(path.Join(templateDir, "*.gohtml")),
	)
}
