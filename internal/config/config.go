package config

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/gi8lino/tiledash/internal/utils"

	"github.com/containeroo/resolver"

	"gopkg.in/yaml.v3"
)

// Default values for customization style.
const (
	defaultGridGap            template.CSS = "2rem"
	defaultGridPadding        template.CSS = "0rem"
	defaultGridMarginTop      template.CSS = "0rem"
	defaultGridMarginBottom   template.CSS = "0rem"
	defaultCardBorderColor    template.CSS = "#ccc"
	defaultCardPadding        template.CSS = "0rem"
	defaultCardBackground     template.CSS = "#fff"
	defaultCardRadius         template.CSS = "0.5rem"
	defaultCardShadow         template.CSS = "0 2px 4px rgba(0, 0, 0, 0.05)"
	defaultHeaderAlign        template.CSS = "left"
	defaultHeaderMarginBottom template.CSS = "0rem"
	defaultFooterMarginTop    template.CSS = "1rem"
	defaultFontFamily         template.CSS = `Segoe UI, sans-serif`
	defaultFontSize           template.CSS = "16px"
)

var illegalCSSChars = []rune{'<', '>', '{', '}', '"', '\'', '`'}

// LoadConfig loads the dashboard configuration from the given path.
func LoadConfig(path string) (DashboardConfig, error) {
	cfg := DashboardConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %v", err)
	}

	r := bytes.NewReader(data)
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true) // fail on unknown YAML keys
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config: %v", err)
	}

	return cfg, nil
}

// ValidateConfig checks the consistency and correctness of a dashboard config and its templates.
// All detected errors (including CSS issues) are returned together in a single error.
func ValidateConfig(cfg *DashboardConfig, tmpl *template.Template) error {
	// Collect all errors and report them together at the end.
	var errs []string

	// Basic grid & refresh checks.
	if cfg.Grid.Columns <= 0 {
		errs = append(errs, "grid.columns must be > 0")
	}
	if cfg.Grid.Rows <= 0 {
		errs = append(errs, "grid.rows must be > 0")
	}
	if cfg.RefreshInterval <= 0 {
		errs = append(errs, "refreshInterval must be > 0")
	}

	// Providers should exist if tiles exist.
	if len(cfg.Tiles) > 0 && len(cfg.Providers) == 0 {
		errs = append(errs, "providers must not be empty when tiles are defined")
	}

	occupied := make(map[[2]int]string)

	for i, tile := range cfg.Tiles {
		label := fmt.Sprintf("tile[%d]", i)
		if tile.Title != "" {
			label += fmt.Sprintf(" (%s)", tile.Title)
		}

		// Title required.
		if strings.TrimSpace(tile.Title) == "" {
			errs = append(errs, fmt.Sprintf("%s: title is required", label))
		}

		// Template must be present, end with .gohtml, and be loaded.
		switch name := tile.Template; {
		case strings.TrimSpace(name) == "":
			errs = append(errs, fmt.Sprintf("%s: template is required", label))
		case !strings.HasSuffix(name, ".gohtml"):
			errs = append(errs, fmt.Sprintf(`%s: template %q must end with ".gohtml"`, label, name))
		case tmpl != nil && tmpl.Lookup(name) == nil:
			// Only check existence if a template set was provided.
			errs = append(errs, fmt.Sprintf(`%s: template %q not found`, label, name))
		}

		// Request validation.
		req := tile.Request
		if strings.TrimSpace(req.Provider) == "" {
			errs = append(errs, fmt.Sprintf("%s: request.provider is required", label))
		} else {
			if _, ok := cfg.Providers[strings.TrimSpace(req.Provider)]; !ok {
				errs = append(errs, fmt.Sprintf("%s: unknown provider %q", label, req.Provider))
			}
		}
		if strings.TrimSpace(req.Path) == "" {
			errs = append(errs, fmt.Sprintf("%s: request.path is required", label))
		}
		if m := strings.ToUpper(strings.TrimSpace(req.Method)); m != "" {
			switch m {
			case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
				// ok
			default:
				errs = append(errs, fmt.Sprintf("%s: request.method %q is not a valid HTTP verb", label, req.Method))
			}
		}

		// Pagination validation.
		if req.Paginate {
			loc := strings.ToUpper(strings.TrimSpace(req.Page.Location))
			if loc != "" && loc != "QUERY" && loc != "BODY" {
				errs = append(errs, fmt.Sprintf("%s: page.location must be 'query' or 'body'", label))
			}
			if req.Page.LimitPages < 0 {
				errs = append(errs, fmt.Sprintf("%s: page.limitPages must be >= 0", label))
			}
		}

		// Position validation (1-based externally; convert to 0-based for bounds).
		row := tile.Position.Row - 1
		col := tile.Position.Col - 1
		colSpan := tile.Position.ColSpan
		if colSpan <= 0 {
			colSpan = 1
		}

		maxRow := cfg.Grid.Rows - 1
		maxCol := cfg.Grid.Columns - 1

		if tile.Position.Row < 1 {
			errs = append(errs, fmt.Sprintf("%s: row %d out of bounds (min 1)", label, tile.Position.Row))
		} else if row > maxRow {
			errs = append(errs, fmt.Sprintf("%s: row %d out of bounds (max %d)", label, tile.Position.Row, cfg.Grid.Rows))
		}
		if tile.Position.Col < 1 {
			errs = append(errs, fmt.Sprintf("%s: col %d out of bounds (min 1)", label, tile.Position.Col))
		} else if col > maxCol {
			errs = append(errs, fmt.Sprintf("%s: col %d out of bounds (max %d)", label, tile.Position.Col, cfg.Grid.Columns))
		}
		if col < 0 {
			errs = append(errs, fmt.Sprintf("%s: colSpan %d out of bounds (min 1)", label, colSpan))
		} else if col+colSpan > cfg.Grid.Columns {
			errs = append(errs, fmt.Sprintf("%s: colSpan %d overflows grid width %d", label, colSpan, cfg.Grid.Columns))
		}

		// Track occupied tiles to detect overlaps.
		if row >= 0 && col >= 0 && col+colSpan <= cfg.Grid.Columns {
			for c := col; c < col+colSpan; c++ {
				key := [2]int{row, c}
				if other, ok := occupied[key]; ok {
					errs = append(errs, fmt.Sprintf("%s: overlaps tile (%d,%d) used by %q",
						label, tile.Position.Row, tile.Position.Col, other))
				}
				occupied[key] = tile.Title
			}
		}

		// Check if provider is defined.
		if _, ok := cfg.Providers[tile.Request.Provider]; !ok {
			errs = append(errs, fmt.Sprintf("%s: unknown provider %q", label, tile.Request.Provider))
		}
	}

	// Always initialize Customization if not provided.
	if cfg.Customization == nil {
		cfg.Customization = &Customization{}
	}

	// Defaults for customization.
	setStyleDefaults(cfg.Customization)

	// Append CSS validation errors.
	errs = append(errs, validateCSSs(cfg.Customization)...)

	// Apply provider defaults (non-breaking) and validate auth blocks.
	setProviderDefaults(cfg)
	errs = append(errs, validateProvidersAuth(cfg)...)

	if len(errs) > 0 {
		return fmt.Errorf("config has errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// setProviderDefaults fills provider-level defaults that are safe and non-breaking.
func setProviderDefaults(cfg *DashboardConfig) {
	for name, p := range cfg.Providers {
		// Default SkipTLSVerify to false when unspecified.
		if p.SkipTLSVerify == nil {
			p.SkipTLSVerify = utils.Ptr(false)
		}
		// Write back because map iteration returns a copy.
		cfg.Providers[name] = p
	}
}

// validateProvidersAuth validates provider auth blocks and returns a list of errors.
// Rules:
//   - At most one auth method may be set (basic XOR bearer).
//   - Basic requires non-empty username and password.
//   - Bearer requires a non-empty token.
func validateProvidersAuth(cfg *DashboardConfig) []string {
	var errs []string
	for name, p := range cfg.Providers {
		hasBasic := p.Auth.Basic != nil
		hasBearer := p.Auth.Bearer != nil

		// Exactly one or zero methods is acceptable; both is an error.
		if hasBasic && hasBearer {
			errs = append(errs, fmt.Sprintf(`provider %q: choose exactly one auth method ("basic" or "bearer"), not both`, name))
			continue // other checks would be redundant
		}

		// Basic auth validation.
		if hasBasic {
			u := strings.TrimSpace(p.Auth.Basic.Username)
			pw := strings.TrimSpace(p.Auth.Basic.Password)
			if u == "" || pw == "" {
				errs = append(errs, fmt.Sprintf(`provider %q: basic auth requires non-empty "username" and "password"`, name))
			} else {
				var err error
				p.Auth.Basic.Username, err = resolver.ResolveVariable(u)
				if err != nil {
					errs = append(errs, fmt.Sprintf("provider %q: basic auth username %q has a not resolvable variable: %v", name, u, err))
				}
				p.Auth.Basic.Password, err = resolver.ResolveVariable(pw)
				if err != nil {
					errs = append(errs, fmt.Sprintf("provider %q: basic auth password %q has a not resolvable variable: %v", name, u, err))
				}
			}
		}

		// Bearer auth validation.
		if hasBearer {
			t := strings.TrimSpace(p.Auth.Bearer.Token)
			if t == "" {
				errs = append(errs, fmt.Sprintf(`provider %q: bearer auth requires non-empty "token"`, name))
			} else {
				var err error
				p.Auth.Bearer.Token, err = resolver.ResolveVariable(t)
				if err != nil {
					errs = append(errs, fmt.Sprintf("provider %q: bearer auth token %q has a not resolvable variable: %v", name, t, err))
				}
			}
		}
	}
	return errs
}

// setDefault assigns dst to val only if *dst is empty.
func setDefault(dst *template.CSS, val template.CSS) {
	if *dst == "" {
		*dst = val
	}
}

// SortCellsByPosition sorts all tiles top-to-bottom, left-to-right.
func (c *DashboardConfig) SortCellsByPosition() {
	sort.SliceStable(c.Tiles, func(i, j int) bool {
		pi := c.Tiles[i].Position
		pj := c.Tiles[j].Position
		if pi.Row != pj.Row {
			return pi.Row < pj.Row
		}
		return pi.Col < pj.Col
	})
}

// setStyleDefaults fills in missing customization fields with default values.
func setStyleDefaults(c *Customization) {
	setDefault(&c.Grid.Gap, defaultGridGap)
	setDefault(&c.Grid.Padding, defaultGridPadding)
	setDefault(&c.Grid.MarginTop, defaultGridMarginTop)
	setDefault(&c.Grid.MarginBottom, defaultGridMarginBottom)

	setDefault(&c.Card.BorderColor, defaultCardBorderColor)
	setDefault(&c.Card.Padding, defaultCardPadding)
	setDefault(&c.Card.BackgroundColor, defaultCardBackground)
	setDefault(&c.Card.BorderRadius, defaultCardRadius)
	setDefault(&c.Card.BoxShadow, defaultCardShadow)

	setDefault(&c.Header.Align, defaultHeaderAlign)
	setDefault(&c.Header.MarginBottom, defaultHeaderMarginBottom)

	setDefault(&c.Footer.MarginTop, defaultFooterMarginTop)

	setDefault(&c.Font.Family, defaultFontFamily)
	setDefault(&c.Font.Size, defaultFontSize)
}

// validateCSSs validates all CSS fields and returns a slice of error messages.
func validateCSSs(customization *Customization) []string {
	var errs []string
	fields := []struct {
		name string
		val  template.CSS
	}{
		{"grid.gap", customization.Grid.Gap},
		{"grid.padding", customization.Grid.Padding},
		{"grid.marginTop", customization.Grid.MarginTop},
		{"grid.marginBottom", customization.Grid.MarginBottom},
		{"card.borderColor", customization.Card.BorderColor},
		{"card.padding", customization.Card.Padding},
		{"card.backgroundColor", customization.Card.BackgroundColor},
		{"card.borderRadius", customization.Card.BorderRadius},
		{"card.boxShadow", customization.Card.BoxShadow},
		{"header.align", customization.Header.Align},
		{"header.marginBottom", customization.Header.MarginBottom},
		{"footer.marginTop", customization.Footer.MarginTop},
		{"font.family", customization.Font.Family},
		{"font.size", customization.Font.Size},
	}

	for _, f := range fields {
		if err := validateCSSValue(f.name, f.val); err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

// validateCSSValue validates a single CSS value.
func validateCSSValue(name string, val template.CSS) error {
	s := strings.TrimSpace(string(val))
	if s == "" {
		return nil
	}
	for _, ch := range illegalCSSChars {
		if strings.ContainsRune(s, ch) {
			return fmt.Errorf("%s: contains illegal character %q", name, ch)
		}
	}
	return nil
}
