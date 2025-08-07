package config

import (
	"html/template"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertCSS(t *testing.T, expected string, actual template.CSS) {
	t.Helper()
	assert.Equal(t, expected, string(actual))
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	t.Run("loads valid YAML file", func(t *testing.T) {
		t.Parallel()

		yaml := `
grid:
  rows: 2
  columns: 2
cells:
  - title: Sample
    query: SELECT *
    template: box
    position: { row: 0, col: 0 }
refreshInterval: 30s
`

		tmp, err := os.CreateTemp("", "test-config-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmp.Name()) // nolint:errcheck

		_, err = tmp.WriteString(yaml)
		require.NoError(t, err)
		require.NoError(t, tmp.Close())

		cfg, err := LoadConfig(tmp.Name())
		require.NoError(t, err)

		assert.Equal(t, 2, cfg.Grid.Rows)
		assert.Equal(t, 2, cfg.Grid.Columns)
		assert.Equal(t, 1, len(cfg.Cells))
		assert.Equal(t, "Sample", cfg.Cells[0].Title)
	})

	t.Run("fails if file missing", func(t *testing.T) {
		t.Parallel()

		_, err := LoadConfig("does-not-exist.yaml")
		assert.Error(t, err)
	})
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid: Grid{
				Rows:    1,
				Columns: 1,
			},
			RefreshInterval: 60,
			Cells: []Cell{
				{
					Title:    "A",
					Query:    "SELECT",
					Template: "card.gohtml",
					Position: Position{Row: 1, Col: 1},
				},
			},
		}

		tmpl := tmplWith(t, "card.gohtml")

		err := ValidateConfig(&cfg, tmpl)
		assert.NoError(t, err)
	})

	t.Run("rejects invalid config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 2},
			RefreshInterval: 0,
			Cells: []Cell{
				{
					Title:    "",
					Query:    "",
					Template: "missing",
					Position: Position{Row: 99, Col: 99},
				},
				{
					Title:    "invalid",
					Query:    "",
					Position: Position{Row: -1, Col: -1},
				},
				{
					Title:    "not-found",
					Query:    "not-found",
					Position: Position{Row: 1, Col: 1},
					Template: "not-fond.gohtml",
				},
			},
		}

		tmpl := tmplWith(t, "almost.gohtml")

		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		expected := []string{
			"  - refreshInterval must be > 0",
			"  - section[0]: title is required",
			"  - section[0]: query is required",
			"  - section[0]: template \"missing\" must end with \".gohtml\"",
			"  - section[0]: row 99 out of bounds (max 1)",
			"  - section[0]: col 99 out of bounds (max 2)",
			"  - section[0]: colSpan 1 overflows grid width 2",
			"  - section[1] (invalid): query is required",
			"  - section[1] (invalid): template is required",
			"  - section[1] (invalid): row -1 out of bounds (min 1)",
			"  - section[1] (invalid): col -1 out of bounds (min 1)",
			"  - section[1] (invalid): colSpan 1 out of bounds (min 1)",
			"  - section[2] (not-found): template \"not-fond.gohtml\" not found",
		}

		assert.EqualError(t, err, "config validation failed:\n"+strings.Join(expected, "\n"))
	})

	t.Run("invalid grid config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 0, Columns: 0},
			RefreshInterval: 0,
		}

		tmpl := tmplWith(t, "only-existing")

		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		expected := []string{
			"  - grid.columns must be > 0",
			"  - grid.rows must be > 0",
			"  - refreshInterval must be > 0",
		}

		assert.EqualError(t, err, "config validation failed:\n"+strings.Join(expected, "\n"))
	})

	t.Run("overlapping cell config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 2},
			RefreshInterval: 2 * time.Second,
			Cells: []Cell{
				{
					Title:    "valid",
					Query:    "valid",
					Template: "valid.gohtml",
					Position: Position{Row: 1, Col: 1},
				},
				{
					Title:    "overlapping",
					Query:    "query",
					Template: "overlapping.gohtml",
					Position: Position{Row: 1, Col: 1},
				},
			},
		}

		tmpl := tmplWith(t, "valid.gohtml", "overlapping.gohtml")

		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		expected := []string{
			"  - section[1] (overlapping): overlaps cell (1,1) used by section \"valid\"",
		}

		assert.EqualError(t, err, "config validation failed:\n"+strings.Join(expected, "\n"))
	})

	t.Run("rejects invalid config with other values", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 2},
			RefreshInterval: 2 * time.Second,
			Cells: []Cell{
				{
					Title:    "almost valid",
					Query:    "some query",
					Template: "almost.gohtml",
					Position: Position{Row: 1, Col: 1, ColSpan: 4},
				},
			},
		}

		tmpl := tmplWith(t, "almost.gohtml")

		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		expected := []string{
			"  - section[0] (almost valid): colSpan 4 overflows grid width 2",
		}

		assert.EqualError(t, err, "config validation failed:\n"+strings.Join(expected, "\n"))
	})

	t.Run("sets style defaults if customization is nil", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 1},
			RefreshInterval: 10 * time.Second,
			Cells: []Cell{
				{
					Title:    "basic",
					Query:    "JQL",
					Template: "default.gohtml",
					Position: Position{Row: 1, Col: 1},
				},
			},
			Customization: nil, // explicitly not set
		}

		tmpl := tmplWith(t, "default.gohtml")
		err := ValidateConfig(&cfg, tmpl)
		require.NoError(t, err)

		// customization should be initialized with default values
		require.NotNil(t, cfg.Customization)
		assert.Equal(t, defaultGridGap, cfg.Customization.Grid.Gap)
		assert.Equal(t, defaultFontSize, cfg.Customization.Font.Size)
	})

	t.Run("fills missing customization fields but preserves set values", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 1},
			RefreshInterval: 30 * time.Second,
			Cells: []Cell{
				{
					Title:    "styled",
					Query:    "Q",
					Template: "t.gohtml",
					Position: Position{Row: 1, Col: 1},
				},
			},
			Customization: &Customization{
				Grid: GridStyle{
					Gap: "4rem",
				},
				Font: FontStyle{
					Family: "Fira Code",
				},
			},
		}

		tmpl := tmplWith(t, "t.gohtml")
		err := ValidateConfig(&cfg, tmpl)
		require.NoError(t, err)

		// Set fields remain
		assertCSS(t, "4rem", cfg.Customization.Grid.Gap)
		assertCSS(t, "Fira Code", cfg.Customization.Font.Family)

		// Unset fields are defaulted
		assert.Equal(t, defaultGridPadding, cfg.Customization.Grid.Padding)
		assert.Equal(t, defaultGridMarginTop, cfg.Customization.Grid.MarginTop)
		assert.Equal(t, defaultFontSize, cfg.Customization.Font.Size)
		assert.Equal(t, defaultCardBackground, cfg.Customization.Card.BackgroundColor)
	})

	t.Run("rejects template without .gohtml extension", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 1},
			RefreshInterval: 10,
			Cells: []Cell{
				{
					Title:    "missing extension",
					Query:    "SELECT",
					Template: "template-without-extension", // no .gohtml
					Position: Position{Row: 1, Col: 1},
				},
			},
		}

		tmpl := tmplWith(t, "template-without-extension.gohtml") // valid template is registered with extension

		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		expected := []string{
			"  - section[0] (missing extension): template \"template-without-extension\" must end with \".gohtml\"",
		}

		assert.EqualError(t, err, "config validation failed:\n"+strings.Join(expected, "\n"))
	})

	t.Run("defaults colSpan to 1 when unset or <= 0", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 2},
			RefreshInterval: 10,
			Cells: []Cell{
				{
					Title:    "default span",
					Query:    "q",
					Template: "default.gohtml",
					Position: Position{Row: 1, Col: 1, ColSpan: 0}, // should default to 1
				},
			},
		}

		tmpl := tmplWith(t, "default.gohtml")
		err := ValidateConfig(&cfg, tmpl)
		require.NoError(t, err)
	})

	t.Run("rejects colSpan that exceeds grid width", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 1, Columns: 2},
			RefreshInterval: 10,
			Cells: []Cell{
				{
					Title:    "wide",
					Query:    "q",
					Template: "wide.gohtml",
					Position: Position{Row: 1, Col: 2, ColSpan: 2},
				},
			},
		}

		tmpl := tmplWith(t, "wide.gohtml")
		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "colSpan 2 overflows grid width 2")
	})

	t.Run("rejects row and col less than 1", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            Grid{Rows: 2, Columns: 2},
			RefreshInterval: 10,
			Cells: []Cell{
				{
					Title:    "zero-based",
					Query:    "q",
					Template: "cell.gohtml",
					Position: Position{Row: 0, Col: 0},
				},
			},
		}

		tmpl := tmplWith(t, "cell.gohtml")
		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		assert.Contains(t, err.Error(), ": row 0 out of bounds")
		assert.Contains(t, err.Error(), ": col 0 out of bounds")
	})
}

// tmplWith returns a template containing the given template names.
func tmplWith(t *testing.T, names ...string) *template.Template {
	t.Helper()
	tmpl := template.New("base")
	for _, name := range names {
		tmpl.New(name).Parse("template " + name) // nolint:errcheck
	}
	return tmpl
}

func TestSetStyleDefaults(t *testing.T) {
	t.Parallel()

	t.Run("applies defaults to empty fields", func(t *testing.T) {
		t.Parallel()
		c := &Customization{}
		setStyleDefaults(c)

		assert.Equal(t, defaultGridGap, c.Grid.Gap)
		assert.Equal(t, defaultGridPadding, c.Grid.Padding)
		assert.Equal(t, defaultGridMarginTop, c.Grid.MarginTop)
		assert.Equal(t, defaultCardBorderColor, c.Card.BorderColor)
		assert.Equal(t, defaultCardPadding, c.Card.Padding)
		assert.Equal(t, defaultCardBackground, c.Card.BackgroundColor)
		assert.Equal(t, defaultCardRadius, c.Card.BorderRadius)
		assert.Equal(t, defaultCardShadow, c.Card.BoxShadow)
		assert.Equal(t, defaultHeaderAlign, c.Header.Align)
		assert.Equal(t, defaultHeaderMarginBottom, c.Header.MarginBottom)
		assert.Equal(t, defaultFontFamily, c.Font.Family)
		assert.Equal(t, defaultFontSize, c.Font.Size)
	})

	t.Run("preserves existing values", func(t *testing.T) {
		t.Parallel()

		c := &Customization{
			Grid: GridStyle{
				Gap:       "2rem",
				Padding:   "3rem",
				MarginTop: "4rem",
			},
			Card: CardStyle{
				BorderColor:     "blue",
				Padding:         "2px",
				BackgroundColor: "#000",
				BorderRadius:    "8px",
				BoxShadow:       "none",
			},
			Header: HeaderStyle{
				Align:        "center",
				MarginBottom: "5rem",
			},
			Font: FontStyle{
				Family: "monospace",
				Size:   "18px",
			},
		}

		setStyleDefaults(c)

		assertCSS(t, "2rem", c.Grid.Gap)
		assertCSS(t, "3rem", c.Grid.Padding)
		assertCSS(t, "4rem", c.Grid.MarginTop)
		assertCSS(t, "blue", c.Card.BorderColor)
		assertCSS(t, "2px", c.Card.Padding)
		assertCSS(t, "#000", c.Card.BackgroundColor)
		assertCSS(t, "8px", c.Card.BorderRadius)
		assertCSS(t, "none", c.Card.BoxShadow)
		assertCSS(t, "center", c.Header.Align)
		assertCSS(t, "5rem", c.Header.MarginBottom)
		assertCSS(t, "monospace", c.Font.Family)
		assertCSS(t, "18px", c.Font.Size)
	})

	t.Run("renders CSS fields without ZgotmplZ or escaping", func(t *testing.T) {
		t.Parallel()

		c := &Customization{
			Font: FontStyle{
				Family: "Segoe UI, sans-serif",
				Size:   "16px",
			},
		}
		setStyleDefaults(c)

		tmpl := template.Must(template.New("test").Parse(`
			<style>
				body {
					font-family: {{ .Font.Family }};
					font-size: {{ .Font.Size }};
				}
			</style>
		`))

		var buf strings.Builder
		err := tmpl.Execute(&buf, c)
		require.NoError(t, err)

		output := buf.String()
		require.NotContains(t, output, "ZgotmplZ", "template rendered unsafe CSS content")
		require.Contains(t, output, "font-family: Segoe UI, sans-serif")
		require.Contains(t, output, "font-size: 16px")
	})
}

func TestSortCellsByPosition(t *testing.T) {
	t.Parallel()

	t.Run("sorts by row then col", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Cells: []Cell{
				{Title: "C", Position: Position{Row: 2, Col: 1}},
				{Title: "A", Position: Position{Row: 1, Col: 2}},
				{Title: "B", Position: Position{Row: 1, Col: 1}},
				{Title: "D", Position: Position{Row: 3, Col: 1}},
			},
		}

		cfg.SortCellsByPosition()

		titles := []string{}
		for _, cell := range cfg.Cells {
			titles = append(titles, cell.Title)
		}

		assert.Equal(t, []string{"B", "A", "C", "D"}, titles)
	})

	t.Run("keeps stable order for equal positions", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Cells: []Cell{
				{Title: "First", Position: Position{Row: 1, Col: 1}},
				{Title: "Second", Position: Position{Row: 1, Col: 1}},
			},
		}

		cfg.SortCellsByPosition()

		assert.Equal(t, "First", cfg.Cells[0].Title)
		assert.Equal(t, "Second", cfg.Cells[1].Title)
	})

	t.Run("sorts empty slice safely", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{}
		cfg.SortCellsByPosition()

		assert.Empty(t, cfg.Cells)
	})
}
