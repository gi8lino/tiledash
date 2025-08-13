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

	t.Run("loads valid YAML file (new schema)", func(t *testing.T) {
		t.Parallel()

		yaml := `
grid:
  rows: 2
  columns: 2
refreshInterval: 30s
tiles:
  - title: Sample
    template: box.gohtml
    position: { row: 1, col: 1 }
    request:
      provider: p
      path: /x
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
		require.Len(t, cfg.Tiles, 1)
		assert.Equal(t, "Sample", cfg.Tiles[0].Title)
	})

	t.Run("fails if file missing", func(t *testing.T) {
		t.Parallel()
		_, err := LoadConfig("does-not-exist.yaml")
		assert.Error(t, err)
	})

	t.Run("rejects unknown keys (KnownFields)", func(t *testing.T) {
		t.Parallel()

		yaml := `
gird:   # typo on purpose
  rows: 1
  columns: 1
refreshInterval: 5s
`
		tmp, err := os.CreateTemp("", "bad-config-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmp.Name()) // nolint:errcheck
		_, _ = tmp.WriteString(yaml)
		_ = tmp.Close()

		_, err = LoadConfig(tmp.Name())
		require.Error(t, err)
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("accepts valid config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 60 * time.Second,
			Providers: map[string]Provider{
				"p": {}, // minimal provider; auth is optional
			},
			Tiles: []Tile{
				{
					Title:    "A",
					Template: "card.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request: Request{
						Provider: "p",
						Method:   "GET",
						Path:     "/ok",
					},
				},
			},
		}

		tmpl := tmplWith(t, "card.gohtml")

		err := cfg.Validate(tmpl)
		assert.NoError(t, err)
	})

	t.Run("rejects invalid config (many errors)", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 2},
			RefreshInterval: 0,                     // triggers refresh error
			Providers:       map[string]Provider{}, // triggers providers empty error
			Tiles: []Tile{
				{
					Title:    "",
					Template: "missing", // no .gohtml
					Position: Position{Row: 99, Col: 99},
					Request:  Request{}, // no provider, no path
				},
				{
					Title:    "invalid",
					Template: "", // required
					Position: Position{Row: -1, Col: -1},
					Request:  Request{}, // no provider, no path
				},
				{
					Title:    "not-found",
					Template: "not-fond.gohtml", // template not in set
					Position: Position{Row: 1, Col: 1},
					Request:  Request{}, // no provider, no path
				},
			},
		}

		tmpl := tmplWith(t, "almost.gohtml")

		err := cfg.Validate(tmpl)
		require.Error(t, err)

		// Note: for empty provider names we expect "request.provider is required"
		// (no extra "unknown provider").
		expected := []string{
			"  - refreshInterval must be > 0",
			"  - providers must not be empty when tiles are defined",

			// tile[0]
			`  - tile[0]: title is required`,
			`  - tile[0]: template "missing" must end with ".gohtml"`,
			`  - tile[0]: request.provider is required`,
			`  - tile[0]: request.path is required`,
			"  - tile[0]: row 99 out of bounds (max 1)",
			"  - tile[0]: col 99 out of bounds (max 2)",
			"  - tile[0]: colSpan 1 overflows grid width 2",

			// tile[1]
			"  - tile[1] (invalid): template is required",
			"  - tile[1] (invalid): request.provider is required",
			"  - tile[1] (invalid): request.path is required",
			"  - tile[1] (invalid): row -1 out of bounds (min 1)",
			"  - tile[1] (invalid): col -1 out of bounds (min 1)",
			"  - tile[1] (invalid): colSpan 1 out of bounds (min 1)",

			// tile[2]
			`  - tile[2] (not-found): template "not-fond.gohtml" not found`,
			"  - tile[2] (not-found): request.provider is required",
			"  - tile[2] (not-found): request.path is required",
		}

		assert.EqualError(t, err, "config has errors:\n"+strings.Join(expected, "\n"))
	})

	t.Run("invalid grid config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 0, Columns: 0},
			RefreshInterval: 0,
		}

		tmpl := tmplWith(t, "only-existing")

		err := cfg.Validate(tmpl)
		require.Error(t, err)

		expected := []string{
			"  - grid.columns must be > 0",
			"  - grid.rows must be > 0",
			"  - refreshInterval must be > 0",
		}

		assert.EqualError(t, err, "config has errors:\n"+strings.Join(expected, "\n"))
	})

	t.Run("overlapping tile config", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 2},
			RefreshInterval: 2 * time.Second,
			Providers: map[string]Provider{
				"p": {},
			},
			Tiles: []Tile{
				{
					Title:    "valid",
					Template: "valid.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "p", Path: "/a"},
				},
				{
					Title:    "overlapping",
					Template: "overlapping.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "p", Path: "/b"},
				},
			},
		}

		tmpl := tmplWith(t, "valid.gohtml", "overlapping.gohtml")

		err := cfg.Validate(tmpl)
		require.Error(t, err)

		expected := []string{
			`  - tile[1] (overlapping): overlaps tile (1,1) used by "valid"`,
		}

		assert.EqualError(t, err, "config has errors:\n"+strings.Join(expected, "\n"))
	})

	t.Run("rejects invalid TTL and HTTP method", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 5 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "bad",
					Template: "bad.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request: Request{
						Provider: "p",
						Method:   "NONSENSE",
						Path:     "/x",
						TTL:      -1,
					},
				},
			},
		}
		tmpl := tmplWith(t, "bad.gohtml")
		err := cfg.Validate(tmpl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `request.method "NONSENSE" is not a valid HTTP verb`)
		assert.Contains(t, err.Error(), "request.ttl must be >= 0")
	})

	t.Run("pagination requires request/response markers", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "p1",
					Template: "p.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request: Request{
						Provider: "p",
						Path:     "/x",
						Paginate: true,
						Page: PageParams{
							Location:   "query",
							ReqStart:   "", // missing
							ReqLimit:   "", // missing
							StartField: "",
							TotalField: "",
						},
					},
				},
			},
		}
		tmpl := tmplWith(t, "p.gohtml")
		err := cfg.Validate(tmpl)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "page.reqStart and page.reqLimit must be set when paginate is true")
		assert.Contains(t, err.Error(), "page.startField or page.totalField should be set")
	})

	t.Run("provider name matching is case-insensitive", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"Jira-V2": {}}, // mixed case in map
			Tiles: []Tile{
				{
					Title:    "ok",
					Template: "ok.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "jira-v2", Path: "/x"},
				},
			},
		}
		tmpl := tmplWith(t, "ok.gohtml")
		err := cfg.Validate(tmpl)
		require.NoError(t, err)
	})

	t.Run("rejects template without .gohtml extension", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "missing extension",
					Template: "template-without-extension", // no .gohtml
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
		}

		tmpl := tmplWith(t, "template-without-extension.gohtml") // registered version
		err := cfg.Validate(tmpl)
		require.Error(t, err)

		expected := []string{
			`  - tile[0] (missing extension): template "template-without-extension" must end with ".gohtml"`,
		}

		assert.EqualError(t, err, "config has errors:\n"+strings.Join(expected, "\n"))
	})

	t.Run("colSpan defaults to 1 when <= 0 (bounds check uses default)", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 2},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "default span",
					Template: "default.gohtml",
					Position: Position{Row: 1, Col: 1, ColSpan: 0}, // treated as 1
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
		}

		tmpl := tmplWith(t, "default.gohtml")
		err := cfg.Validate(tmpl)
		require.NoError(t, err)
	})

	t.Run("rejects colSpan that exceeds grid width", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 2},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "wide",
					Template: "wide.gohtml",
					Position: Position{Row: 1, Col: 2, ColSpan: 2},
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
		}

		tmpl := tmplWith(t, "wide.gohtml")
		err := cfg.Validate(tmpl)
		require.Error(t, err)

		assert.Contains(t, err.Error(), "colSpan 2 overflows grid width 2")
	})

	t.Run("rejects row and col less than 1", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 2, Columns: 2},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "zero-based",
					Template: "tile.gohtml",
					Position: Position{Row: 0, Col: 0},
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
		}

		tmpl := tmplWith(t, "tile.gohtml")
		err := cfg.Validate(tmpl)
		require.Error(t, err)

		assert.Contains(t, err.Error(), ": row 0 out of bounds")
		assert.Contains(t, err.Error(), ": col 0 out of bounds")
	})

	t.Run("sets style defaults if customization is nil", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 10 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "basic",
					Template: "default.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
			Customization: nil, // explicitly not set
		}

		tmpl := tmplWith(t, "default.gohtml")
		err := cfg.Validate(tmpl)
		require.NoError(t, err)

		require.NotNil(t, cfg.Customization)
		assert.Equal(t, defaultGridGap, cfg.Customization.Grid.Gap)
		assert.Equal(t, defaultFontSize, cfg.Customization.Font.Size)
	})

	t.Run("fills missing customization fields but preserves set values", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 30 * time.Second,
			Providers:       map[string]Provider{"p": {}},
			Tiles: []Tile{
				{
					Title:    "styled",
					Template: "t.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
			Customization: &Customization{
				Grid: CustomGrid{
					Gap: template.CSS("4rem"),
				},
				Font: CustomFont{
					Family: template.CSS("Fira Code"),
				},
			},
		}

		tmpl := tmplWith(t, "t.gohtml")
		err := cfg.Validate(tmpl)
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
}

func TestResolveProvidersAuth(t *testing.T) {
	//	t.Parallel()  // Paralell inpossible due GetEnv

	t.Run("resolves bearer token", func(t *testing.T) {
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"jira": {Auth: AuthConfig{Bearer: &BearerAuth{Token: "env:JIRA_TOKEN"}}},
			},
		}
		t.Setenv("JIRA_TOKEN", "secret123")

		err := cfg.ResolveProvidersAuth()
		require.NoError(t, err)
		assert.Equal(t, "secret123", cfg.Providers["jira"].Auth.Bearer.Token)
	})

	t.Run("resolves basic username/password", func(t *testing.T) {
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"svc": {Auth: AuthConfig{Basic: &BasicAuth{
					Username: "env:USER_X",
					Password: "env:PASS_X",
				}}},
			},
		}
		t.Setenv("USER_X", "bob@example.com")
		t.Setenv("PASS_X", "ap1-t0ken")

		err := cfg.ResolveProvidersAuth()
		require.NoError(t, err)
		b := cfg.Providers["svc"].Auth.Basic
		require.NotNil(t, b)
		assert.Equal(t, "bob@example.com", b.Username)
		assert.Equal(t, "ap1-t0ken", b.Password)
	})

	t.Run("aggregates resolution errors", func(t *testing.T) {
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"p1": {Auth: AuthConfig{Bearer: &BearerAuth{Token: "env:MISSING1"}}},
				"p2": {Auth: AuthConfig{Basic: &BasicAuth{
					Username: "env:MISSING2",
					Password: "env:MISSING3",
				}}},
			},
		}
		err := cfg.ResolveProvidersAuth()
		require.Error(t, err)
		msg := err.Error()
		assert.Contains(t, msg, `provider auth has errors:`)
		assert.Contains(t, msg, `provider "p1": bearer auth token "env:MISSING1" is not resolvable`)
		assert.Contains(t, msg, `provider "p2": basic auth username "env:MISSING2" is not resolvable`)
		assert.Contains(t, msg, `provider "p2": basic auth password "env:MISSING3" is not resolvable`)
	})

	t.Run("Validate does not resolve env placeholders", func(t *testing.T) {
		cfg := DashboardConfig{
			Grid:            &GridConfig{Rows: 1, Columns: 1},
			RefreshInterval: 5 * time.Second,
			Providers: map[string]Provider{
				"p": {Auth: AuthConfig{Bearer: &BearerAuth{Token: "env:TOK"}}},
			},
			Tiles: []Tile{
				{
					Title:    "ok",
					Template: "ok.gohtml",
					Position: Position{Row: 1, Col: 1},
					Request:  Request{Provider: "p", Path: "/x"},
				},
			},
		}
		tmpl := tmplWith(t, "ok.gohtml")
		err := cfg.Validate(tmpl)
		require.NoError(t, err)
		// still unresolved here
		assert.Equal(t, "env:TOK", cfg.Providers["p"].Auth.Bearer.Token)

		t.Setenv("TOK", "resolved")
		require.NoError(t, cfg.ResolveProvidersAuth())
		assert.Equal(t, "resolved", cfg.Providers["p"].Auth.Bearer.Token)
	})
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
			Grid: CustomGrid{
				Gap:       template.CSS("2rem"),
				Padding:   template.CSS("3rem"),
				MarginTop: template.CSS("4rem"),
			},
			Card: CustomCard{
				BorderColor:     template.CSS("blue"),
				Padding:         template.CSS("2px"),
				BackgroundColor: template.CSS("#000"),
				BorderRadius:    template.CSS("8px"),
				BoxShadow:       template.CSS("none"),
			},
			Header: CustomHeader{
				Align:        template.CSS("center"),
				MarginBottom: template.CSS("5rem"),
			},
			Font: CustomFont{
				Family: template.CSS("monospace"),
				Size:   template.CSS("18px"),
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
			Font: CustomFont{
				Family: template.CSS("Segoe UI, sans-serif"),
				Size:   template.CSS("16px"),
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
			Tiles: []Tile{
				{Title: "C", Position: Position{Row: 2, Col: 1}},
				{Title: "A", Position: Position{Row: 1, Col: 2}},
				{Title: "B", Position: Position{Row: 1, Col: 1}},
				{Title: "D", Position: Position{Row: 3, Col: 1}},
			},
		}

		cfg.SortCellsByPosition()

		titles := make([]string, 0, len(cfg.Tiles))
		for _, tile := range cfg.Tiles {
			titles = append(titles, tile.Title)
		}

		assert.Equal(t, []string{"B", "A", "C", "D"}, titles)
	})

	t.Run("keeps stable order for equal positions", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{
			Tiles: []Tile{
				{Title: "First", Position: Position{Row: 1, Col: 1}},
				{Title: "Second", Position: Position{Row: 1, Col: 1}},
			},
		}

		cfg.SortCellsByPosition()

		assert.Equal(t, "First", cfg.Tiles[0].Title)
		assert.Equal(t, "Second", cfg.Tiles[1].Title)
	})

	t.Run("sorts empty slice safely", func(t *testing.T) {
		t.Parallel()

		cfg := DashboardConfig{}
		cfg.SortCellsByPosition()

		assert.Empty(t, cfg.Tiles)
	})
}

func TestValidateCSSValue(t *testing.T) {
	t.Parallel()

	t.Run("accepts empty value", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateCSSValue("font.size", ""))
	})

	t.Run("accepts normal values", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validateCSSValue("grid.gap", template.CSS("2rem")))
		require.NoError(t, validateCSSValue("card.borderColor", template.CSS("#ccc")))
		require.NoError(t, validateCSSValue("font.family", template.CSS("Segoe UI, sans-serif")))
	})

	t.Run("rejects illegal characters", func(t *testing.T) {
		t.Parallel()
		err := validateCSSValue("font.family", template.CSS(`bad"value`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `font.family: contains illegal character '"'`)

		err = validateCSSValue("grid.gap", template.CSS("2rem<"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), `grid.gap: contains illegal character '<'`)
	})
}

func TestValidateCSSs(t *testing.T) {
	t.Parallel()

	t.Run("no errors for sane customization", func(t *testing.T) {
		t.Parallel()
		c := &Customization{
			Grid: CustomGrid{
				Gap:          template.CSS("2rem"),
				Padding:      template.CSS("0rem"),
				MarginTop:    template.CSS("1rem"),
				MarginBottom: template.CSS("1rem"),
			},
			Card: CustomCard{
				BorderColor:     template.CSS("#ccc"),
				Padding:         template.CSS("1rem"),
				BackgroundColor: template.CSS("#fff"),
				BorderRadius:    template.CSS("0.5rem"),
				BoxShadow:       template.CSS("0 2px 4px rgba(0,0,0,0.05)"),
			},
			Header: CustomHeader{
				Align:        template.CSS("center"),
				MarginBottom: template.CSS("0.5rem"),
			},
			Footer: CustomFooter{
				MarginTop: template.CSS("1rem"),
			},
			Font: CustomFont{
				Family: template.CSS("Segoe UI, sans-serif"),
				Size:   template.CSS("16px"),
			},
		}
		setStyleDefaults(c)
		errs := validateCSSs(c)
		require.Empty(t, errs)
	})

	t.Run("collects all illegal char errors", func(t *testing.T) {
		t.Parallel()
		c := &Customization{
			Grid: CustomGrid{
				Gap: template.CSS("2rem"), // ok
			},
			Card: CustomCard{
				BorderColor: template.CSS(`#fff"`), // illegal quote
			},
			Font: CustomFont{
				Family: template.CSS("monospace<"), // illegal <
				Size:   template.CSS("16px"),       // ok
			},
		}
		errs := validateCSSs(c)
		require.Len(t, errs, 2)
		text := strings.Join(errs, "\n")
		assert.Contains(t, text, `card.borderColor: contains illegal character '"'`)
		assert.Contains(t, text, `font.family: contains illegal character '<'`)
	})
}

// Test setProviderDefaults behavior.
func TestSetProviderDefaults(t *testing.T) {
	t.Parallel()

	t.Run("defaults SkipTLSVerify to false", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"p1": {}, // SkipTLSVerify is nil
			},
		}
		setProviderDefaults(&cfg)
		if cfg.Providers["p1"].SkipTLSVerify == nil || *cfg.Providers["p1"].SkipTLSVerify != false {
			t.Fatalf("SkipTLSVerify default not applied")
		}
	})
}

func TestValidateProvidersAuth(t *testing.T) {
	t.Parallel()

	t.Run("both basic and bearer set", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"p": {
					Auth: AuthConfig{
						Basic:  &BasicAuth{Username: "u", Password: "p"},
						Bearer: &BearerAuth{Token: "t"},
					},
				},
			},
		}
		errs := validateProvidersAuth(&cfg)
		if len(errs) != 1 {
			t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
		}
	})

	t.Run("basic missing username or password", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"p1": {Auth: AuthConfig{Basic: &BasicAuth{Username: "", Password: "x"}}},
				"p2": {Auth: AuthConfig{Basic: &BasicAuth{Username: "x", Password: ""}}},
			},
		}
		errs := validateProvidersAuth(&cfg)
		assert.Len(t, errs, 2)
	})

	t.Run("bearer missing token", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"p": {Auth: AuthConfig{Bearer: &BearerAuth{Token: ""}}},
			},
		}
		errs := validateProvidersAuth(&cfg)
		assert.Len(t, errs, 1)
	})

	t.Run("no auth or single valid method is OK", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Providers: map[string]Provider{
				"none":  {},
				"basic": {Auth: AuthConfig{Basic: &BasicAuth{Username: "u", Password: "p"}}},
				"bear":  {Auth: AuthConfig{Bearer: &BearerAuth{Token: "t"}}},
			},
		}
		errs := validateProvidersAuth(&cfg)
		assert.Len(t, errs, 0)
	})
}

// tmplWith returns a template containing the given template names.
func tmplWith(t *testing.T, names ...string) *template.Template {
	t.Helper()
	tmpl := template.New("base")
	for _, name := range names {
		template.Must(tmpl.New(name).Parse("template " + name))
	}
	return tmpl
}
