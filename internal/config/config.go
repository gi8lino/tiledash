package config

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"

	"github.com/containeroo/resolver"
	"github.com/gi8lino/tiledash/internal/utils"
	"gopkg.in/yaml.v3"
)

// Defaults for CSS customization. Values are safe and non-breaking if omitted.
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

// illegalCSSChars are disallowed to reduce the risk of injecting invalid or unsafe CSS.
var illegalCSSChars = []rune{'<', '>', '{', '}', '"', '\'', '`'}

// LoadConfig reads and unmarshals the dashboard YAML at path into a DashboardConfig.
// Unknown YAML keys are rejected (yaml.Decoder.KnownFields(true)).
func LoadConfig(path string) (DashboardConfig, error) {
	var cfg DashboardConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}

	r := bytes.NewReader(data)
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true) // fail on unknown YAML keys

	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}

// Validate performs **structural** checks on the configuration (grid, tiles, providers,
// templates, pagination wiring) and sets harmless defaults (e.g., CSS, SkipTLSVerify).
//
// This method does NOT resolve environment variables/placeholders in provider auth.
// Call ResolveProvidersAuth() after Validate() to resolve secrets.
//
// If any problems are found, a single aggregated error is returned.
func (cfg *DashboardConfig) Validate(tmpl *template.Template) error {
	var errs []string

	// Grid/tiles/template/request shape
	errs = append(errs, validateGridAndTiles(cfg, tmpl)...)

	// Ensure customization exists + apply CSS defaults, then validate CSS values
	if cfg.Customization == nil {
		cfg.Customization = &Customization{}
	}
	setStyleDefaults(cfg.Customization)
	errs = append(errs, validateCSSs(cfg.Customization)...)

	// Provider-side structural checks and defaulting (no secret resolution here)
	errs = append(errs, validateProvidersAuth(cfg)...)
	setProviderDefaults(cfg)

	if len(errs) > 0 {
		return fmt.Errorf("config has errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// ResolveProvidersAuth resolves environment variable placeholders (e.g. "env:FOO")
// for provider authentication (basic/bearer) and writes the resolved values back.
//
// Resolution happens after Validate so we only resolve for providers/fields that
// are structurally valid. If resolution fails for any provider, an aggregated error
// is returned and partial resolutions (for other providers) may have already been applied.
func (cfg *DashboardConfig) ResolveProvidersAuth() error {
	var errs []string

	for name, p := range cfg.Providers {
		// Work on a local copy (map iteration returns a copy); write back once.
		pp := p

		// Basic auth
		if pp.Auth.Basic != nil {
			if u := strings.TrimSpace(pp.Auth.Basic.Username); u != "" {
				if ru, err := resolver.ResolveVariable(u); err != nil {
					errs = append(errs, fmt.Sprintf(`provider %q: basic auth username %q is not resolvable: %v`, name, u, err))
				} else {
					pp.Auth.Basic.Username = ru
				}
			}
			if pw := strings.TrimSpace(pp.Auth.Basic.Password); pw != "" {
				if rp, err := resolver.ResolveVariable(pw); err != nil {
					errs = append(errs, fmt.Sprintf(`provider %q: basic auth password %q is not resolvable: %v`, name, pw, err))
				} else {
					pp.Auth.Basic.Password = rp
				}
			}
		}

		// Bearer auth
		if pp.Auth.Bearer != nil {
			if t := strings.TrimSpace(pp.Auth.Bearer.Token); t != "" {
				if rt, err := resolver.ResolveVariable(t); err != nil {
					errs = append(errs, fmt.Sprintf(`provider %q: bearer auth token %q is not resolvable: %v`, name, t, err))
				} else {
					pp.Auth.Bearer.Token = rt
				}
			}
		}

		cfg.Providers[name] = pp // single write-back per provider
	}

	if len(errs) > 0 {
		return fmt.Errorf("provider auth has errors:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// SortCellsByPosition orders tiles top-to-bottom then left-to-right (1-based positions).
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

// validateGridAndTiles performs structural checks for grid/tiles/template/request/pagination.
func validateGridAndTiles(cfg *DashboardConfig, tmpl *template.Template) []string {
	var errs []string

	// Basic grid & refresh constraints
	if cfg.Grid.Columns <= 0 {
		errs = append(errs, "grid.columns must be > 0")
	}
	if cfg.Grid.Rows <= 0 {
		errs = append(errs, "grid.rows must be > 0")
	}
	if cfg.RefreshInterval <= 0 {
		errs = append(errs, "refreshInterval must be > 0")
	}

	// At least one provider is required to serve tiles
	if len(cfg.Tiles) > 0 && len(cfg.Providers) == 0 {
		errs = append(errs, "providers must not be empty when tiles are defined")
	}

	// Build a case-insensitive index of providers for request validation
	provIndex := make(map[string]Provider, len(cfg.Providers))
	for name, p := range cfg.Providers {
		key := strings.ToLower(strings.TrimSpace(name))
		provIndex[key] = p
	}

	occupied := make(map[[2]int]string) // detect overlaps

	for i, tile := range cfg.Tiles {
		label := fmt.Sprintf("tile[%d]", i)
		if t := strings.TrimSpace(tile.Title); t != "" {
			label += fmt.Sprintf(" (%s)", t)
		}

		// Title
		if strings.TrimSpace(tile.Title) == "" {
			errs = append(errs, fmt.Sprintf("%s: title is required", label))
		}

		// Template name must end with .gohtml and exist (if template set provided)
		switch name := strings.TrimSpace(tile.Template); {
		case name == "":
			errs = append(errs, fmt.Sprintf("%s: template is required", label))
		case !strings.HasSuffix(name, ".gohtml"):
			errs = append(errs, fmt.Sprintf(`%s: template %q must end with ".gohtml"`, label, name))
		case tmpl != nil && tmpl.Lookup(name) == nil:
			errs = append(errs, fmt.Sprintf(`%s: template %q not found`, label, name))
		}

		// Request/provider/method/path/ttl shape
		req := tile.Request
		provName := strings.ToLower(strings.TrimSpace(req.Provider))
		if provName == "" {
			errs = append(errs, fmt.Sprintf("%s: request.provider is required", label))
		} else if _, ok := provIndex[provName]; !ok {
			errs = append(errs, fmt.Sprintf("%s: unknown provider %q", label, req.Provider))
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

		if req.TTL < 0 {
			errs = append(errs, fmt.Sprintf("%s: request.ttl must be >= 0", label))
		}

		// Pagination wiring (only when enabled)
		if req.Paginate {
			loc := strings.ToUpper(strings.TrimSpace(req.Page.Location))
			if loc != "" && loc != "QUERY" && loc != "BODY" {
				errs = append(errs, fmt.Sprintf("%s: page.location must be 'query' or 'body'", label))
			}
			if req.Page.LimitPages < 0 {
				errs = append(errs, fmt.Sprintf("%s: page.limitPages must be >= 0", label))
			}
			// Request field names are required if we send subsequent paginated requests
			if strings.TrimSpace(req.Page.ReqStart) == "" || strings.TrimSpace(req.Page.ReqLimit) == "" {
				errs = append(errs, fmt.Sprintf("%s: page.reqStart and page.reqLimit must be set when paginate is true", label))
			}
			// At least one response marker to detect progress/end (start or total)
			if strings.TrimSpace(req.Page.StartField) == "" && strings.TrimSpace(req.Page.TotalField) == "" {
				errs = append(errs, fmt.Sprintf("%s: page.startField or page.totalField should be set to detect pagination progress", label))
			}
		}

		// Position (input is 1-based; convert to 0-based for bounds checks)
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

		// Overlap detection across occupied cells
		if row >= 0 && col >= 0 && col+colSpan <= cfg.Grid.Columns {
			for c := col; c < col+colSpan; c++ {
				key := [2]int{row, c}
				if other, ok := occupied[key]; ok {
					errs = append(errs, fmt.Sprintf(
						"%s: overlaps tile (%d,%d) used by %q", label, tile.Position.Row, tile.Position.Col, other))
				}
				occupied[key] = tile.Title
			}
		}
	}
	return errs
}

// setProviderDefaults writes safe defaults for providers (e.g., SkipTLSVerify=false when omitted).
func setProviderDefaults(cfg *DashboardConfig) {
	for name, p := range cfg.Providers {
		if p.SkipTLSVerify == nil {
			p.SkipTLSVerify = utils.Ptr(false)
		}
		cfg.Providers[name] = p // write back because ranging maps yields a copy
	}
}

// validateProvidersAuth only checks **shape** of auth blocks (mutually exclusive and non-empty fields).
// It does not resolve environment variables; call ResolveProvidersAuth for that.
func validateProvidersAuth(cfg *DashboardConfig) []string {
	var errs []string

	for name, p := range cfg.Providers {
		hasBasic := p.Auth.Basic != nil
		hasBearer := p.Auth.Bearer != nil

		if hasBasic && hasBearer {
			errs = append(errs, fmt.Sprintf(
				`provider %q: choose exactly one auth method ("basic" or "bearer"), not both`, name))
			continue
		}

		if hasBasic {
			u := strings.TrimSpace(p.Auth.Basic.Username)
			pw := strings.TrimSpace(p.Auth.Basic.Password)
			if u == "" || pw == "" {
				errs = append(errs, fmt.Sprintf(
					`provider %q: basic auth requires non-empty "username" and "password"`, name))
			}
		}

		if hasBearer {
			t := strings.TrimSpace(p.Auth.Bearer.Token)
			if t == "" {
				errs = append(errs, fmt.Sprintf(
					`provider %q: bearer auth requires non-empty "token"`, name))
			}
		}
	}
	return errs
}

// setDefault assigns val to dst only if dst is currently empty.
func setDefault(dst *template.CSS, val template.CSS) {
	if *dst == "" {
		*dst = val
	}
}

// setStyleDefaults populates missing customization fields with safe defaults.
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

// validateCSSs validates all CSS customization fields and aggregates errors.
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

// validateCSSValue checks a single CSS value for disallowed characters.
// Empty values are allowed (treated as "not set").
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
