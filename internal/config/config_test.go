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
					Template: "card",
					Position: Position{Row: 0, Col: 0},
				},
			},
		}

		tmpl := tmplWith(t, "card")

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
			},
		}

		tmpl := tmplWith(t, "only-existing")

		err := ValidateConfig(&cfg, tmpl)
		require.Error(t, err)

		expected := []string{
			"  - refreshInterval must be > 0",
			"  - section[0]: title is required",
			"  - section[0]: query is required",
			`  - section[0]: template "missing" not found`,
			"  - section[0]: row 99 out of bounds (max 0)",
			"  - section[0]: col 99 out of bounds (max 1)",
			"  - section[0]: colSpan 1 overflows grid width 2",
			"  - section[1] (invalid): query is required",
			"  - section[1] (invalid): template is required",
			"  - section[1] (invalid): row -1 out of bounds (max 0)", // ← this was missing
			"  - section[1] (invalid): col -1 out of bounds (max 1)",
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
					Template: "almost",
					Position: Position{Row: 0, Col: 1, ColSpan: 4},
				},
			},
		}

		tmpl := tmplWith(t, "almost")

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
					Template: "default",
					Position: Position{Row: 0, Col: 0},
				},
			},
			Customization: nil, // explicitly not set
		}

		tmpl := tmplWith(t, "default")
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
					Template: "t",
					Position: Position{Row: 0, Col: 0},
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

		tmpl := tmplWith(t, "t")
		err := ValidateConfig(&cfg, tmpl)
		require.NoError(t, err)

		// Set fields remain
		assert.Equal(t, "4rem", cfg.Customization.Grid.Gap)
		assert.Equal(t, "Fira Code", cfg.Customization.Font.Family)

		// Unset fields are defaulted
		assert.Equal(t, defaultGridPadding, cfg.Customization.Grid.Padding)
		assert.Equal(t, defaultGridMarginTop, cfg.Customization.Grid.MarginTop)
		assert.Equal(t, defaultFontSize, cfg.Customization.Font.Size)
		assert.Equal(t, defaultCardBackground, cfg.Customization.Card.BackgroundColor)
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

		assert.Equal(t, "2rem", c.Grid.Gap)
		assert.Equal(t, "3rem", c.Grid.Padding)
		assert.Equal(t, "4rem", c.Grid.MarginTop)
		assert.Equal(t, "blue", c.Card.BorderColor)
		assert.Equal(t, "2px", c.Card.Padding)
		assert.Equal(t, "#000", c.Card.BackgroundColor)
		assert.Equal(t, "8px", c.Card.BorderRadius)
		assert.Equal(t, "none", c.Card.BoxShadow)
		assert.Equal(t, "center", c.Header.Align)
		assert.Equal(t, "5rem", c.Header.MarginBottom)
		assert.Equal(t, "monospace", c.Font.Family)
		assert.Equal(t, "18px", c.Font.Size)
	})
}
