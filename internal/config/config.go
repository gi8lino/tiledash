package config

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Default values for customization style
const (
	defaultGridGap            string = "2rem"
	defaultGridPadding        string = "0rem"
	defaultGridMarginTop      string = "0rem"
	defaultGridMarginBottom   string = "0rem"
	defaultCardBorderColor    string = "#ccc"
	defaultCardPadding        string = "0rem"
	defaultCardBackground     string = "#fff"
	defaultCardRadius         string = "0.5rem"
	defaultCardShadow         string = "0 2px 4px rgba(0, 0, 0, 0.05)"
	defaultHeaderAlign        string = "left"
	defaultHeaderMarginBottom string = "0rem"
	defaultFooterMarginTop    string = "1rem"
	defaultFontFamily         string = `Segoe UI, sans-serif`
	defaultFontSize           string = "16px"
)

// LoadConfig loads the dashboard configuration from the given path.
func LoadConfig(path string) (DashboardConfig, error) {
	cfg := DashboardConfig{}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("invalid config: %v", err)
	}

	return cfg, nil
}

// ValidateConfig checks the consistency and correctness of a dashboard config and its templates.
func ValidateConfig(cfg *DashboardConfig, tmpl *template.Template) error {
	var errs []string

	if cfg.Grid.Columns <= 0 {
		errs = append(errs, "grid.columns must be > 0")
	}
	if cfg.Grid.Rows <= 0 {
		errs = append(errs, "grid.rows must be > 0")
	}
	if cfg.RefreshInterval <= 0 {
		errs = append(errs, "refreshInterval must be > 0")
	}

	occupied := make(map[[2]int]string)

	for i, section := range cfg.Cells {
		label := fmt.Sprintf("section[%d]", i)
		if section.Title != "" {
			label += fmt.Sprintf(" (%s)", section.Title)
		}

		if section.Title == "" {
			errs = append(errs, fmt.Sprintf("%s: title is required", label))
		}
		if section.Query == "" {
			errs = append(errs, fmt.Sprintf("%s: query is required", label))
		}
		if section.Template == "" {
			errs = append(errs, fmt.Sprintf("%s: template is required", label))
		} else if tmpl.Lookup(section.Template) == nil {
			errs = append(errs, fmt.Sprintf(`%s: template %q not found`, label, section.Template))
		}

		// colSpan default
		pos := section.Position
		colSpan := pos.ColSpan
		if colSpan <= 0 {
			colSpan = 1
		}

		// Safe bounds
		maxRow := max(cfg.Grid.Rows-1, 0)
		maxCol := max(cfg.Grid.Columns-1, 0)

		if pos.Row < 0 || pos.Row > maxRow {
			errs = append(errs, fmt.Sprintf("%s: row %d out of bounds (max %d)", label, pos.Row, maxRow))
		}
		if pos.Col < 0 || pos.Col > maxCol {
			errs = append(errs, fmt.Sprintf("%s: col %d out of bounds (max %d)", label, pos.Col, maxCol))
		}
		if pos.Col+colSpan > cfg.Grid.Columns {
			errs = append(errs, fmt.Sprintf("%s: colSpan %d overflows grid width %d", label, colSpan, cfg.Grid.Columns))
		}

		// Only track occupied cells if position is in bounds
		if pos.Row >= 0 && pos.Col >= 0 && pos.Col+colSpan <= cfg.Grid.Columns {
			for c := pos.Col; c < pos.Col+colSpan; c++ {
				key := [2]int{pos.Row, c}
				if other, ok := occupied[key]; ok {
					errs = append(errs, fmt.Sprintf("%s: overlaps cell (%d,%d) used by section %q", label, pos.Row, c, other))
				}
				occupied[key] = section.Title
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}

	// Always initialize Customization if not provided
	if cfg.Customization == nil {
		cfg.Customization = &Customization{}
	}

	setStyleDefaults(cfg.Customization)

	return nil
}

// setDefault assigns dst to val only if *dst is empty.
func setDefault(dst *string, val string) {
	if *dst == "" {
		*dst = val
	}
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
