package templates_test

import (
	"html/template"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/gi8lino/jirapanel/internal/templates"
	"github.com/gi8lino/jirapanel/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBaseTemplates(t *testing.T) {
	t.Parallel()

	t.Run("parses real base templates successfully", func(t *testing.T) {
		t.Parallel()

		webFS := os.DirFS("../../") // show root directory

		funcMap := templates.TemplateFuncMap()
		baseTmpl := templates.ParseBaseTemplates(webFS, funcMap)
		require.NotNil(t, baseTmpl)

		base := baseTmpl.Lookup("base")
		require.NotNil(t, base)

		for _, name := range []string{"base", "footer", "error"} {
			assert.NotNil(t, baseTmpl.Lookup(name), "template %q should be parsed", name)
		}
	})

	t.Run("parses all base templates successfully", func(t *testing.T) {
		t.Parallel()

		// Provide dummy base, footer, and error templates
		webFS := fstest.MapFS{
			"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}base{{end}}`)},
			"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}error{{end}}`)},
		}

		tmpl := templates.ParseBaseTemplates(webFS, template.FuncMap{})
		assert.NotNil(t, tmpl)

		// Ensure all defined templates exist
		for _, name := range []string{"base", "footer", "error"} {
			assert.NotNil(t, tmpl.Lookup(name), "template %q should be parsed", name)
		}
	})

	t.Run("fails if template is broken", func(t *testing.T) {
		// Provide dummy base, footer, and error templates
		webFS := fstest.MapFS{
			"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}{{ if .Title }} <h1>{{ .Title }}</h1>`)}, // missing end
			"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}footer{{end}}`)},
			"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}error{{end}}`)},
		}

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic due to invalid template, but got none")
			}
		}()

		tmpl := templates.ParseBaseTemplates(webFS, template.FuncMap{})
		assert.NotNil(t, tmpl)
	})
}

func TestParseSectionTemplates(t *testing.T) {
	t.Parallel()

	t.Run("parses all templates in section directory", func(t *testing.T) {
		t.Parallel()

		// Create a temp directory with valid section templates
		dir := t.TempDir()

		testutils.MustWriteFile(t, filepath.Join(dir, "foo.gohtml"), `{{define "foo"}}foo content{{end}}`)
		testutils.MustWriteFile(t, filepath.Join(dir, "bar.gohtml"), `{{define "bar"}}bar content{{end}}`)

		tmpl, err := templates.ParseSectionTemplates(dir, template.FuncMap{})
		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.NotNil(t, tmpl.Lookup("foo"))
		assert.NotNil(t, tmpl.Lookup("bar"))
	})

	t.Run("returns empty template when no files match", func(t *testing.T) {
		t.Parallel()

		emptyDir := t.TempDir()

		tmpl, err := templates.ParseSectionTemplates(emptyDir, template.FuncMap{})
		assert.NoError(t, err)
		assert.NotNil(t, tmpl)
		assert.Nil(t, tmpl.Lookup("any"))
	})

	t.Run("panics if parsing fails", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()

		// Write malformed template
		testutils.MustWriteFile(t, filepath.Join(dir, "bad.gohtml"), `{{define "bad"}}{{end}`) // unclosed

		_, err := templates.ParseSectionTemplates(dir, template.FuncMap{})
		assert.Error(t, err)
		assert.EqualError(t, err, "template: bad.gohtml:1: bad character U+007D '}'")
	})
}
