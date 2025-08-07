package config

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Default values for customization style
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

		switch name := section.Template; {
		case name == "":
			errs = append(errs, fmt.Sprintf("%s: template is required", label))
		case !strings.HasSuffix(name, ".gohtml"):
			errs = append(errs, fmt.Sprintf(`%s: template %q must end with ".gohtml"`, label, name))
		case tmpl.Lookup(name) == nil:
			errs = append(errs, fmt.Sprintf(`%s: template %q not found`, label, name))
		}

		// Convert from 1-based to 0-based indexing
		row := section.Position.Row - 1
		col := section.Position.Col - 1
		colSpan := section.Position.ColSpan
		if colSpan <= 0 {
			colSpan = 1
		}

		// Bounds for validation
		maxRow := cfg.Grid.Rows - 1
		maxCol := cfg.Grid.Columns - 1

		// Validate positions in user-friendly 1-based terms
		if section.Position.Row < 1 {
			errs = append(errs, fmt.Sprintf("%s: row %d out of bounds (min 1)", label, section.Position.Row))
		} else if row > maxRow {
			errs = append(errs, fmt.Sprintf("%s: row %d out of bounds (max %d)", label, section.Position.Row, cfg.Grid.Rows))
		}
		if section.Position.Col < 1 {
			errs = append(errs, fmt.Sprintf("%s: col %d out of bounds (min 1)", label, section.Position.Col))
		} else if col > maxCol {
			errs = append(errs, fmt.Sprintf("%s: col %d out of bounds (max %d)", label, section.Position.Col, cfg.Grid.Columns))
		}
		if col < 0 {
			errs = append(errs, fmt.Sprintf("%s: colSpan %d out of bounds (min 1)", label, colSpan))
		} else if col+colSpan > cfg.Grid.Columns {
			errs = append(errs, fmt.Sprintf("%s: colSpan %d overflows grid width %d", label, colSpan, cfg.Grid.Columns))
		}

		// Only track occupied if valid
		if row >= 0 && col >= 0 && col+colSpan <= cfg.Grid.Columns {
			for c := col; c < col+colSpan; c++ {
				key := [2]int{row, c}
				if other, ok := occupied[key]; ok {
					errs = append(errs, fmt.Sprintf("%s: overlaps cell (%d,%d) used by section %q", label, section.Position.Row, section.Position.Col, other))
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
func setDefault(dst *template.CSS, val template.CSS) {
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

// SortCellsByPosition sorts all cells top-to-bottom, left-to-right.
func (c *DashboardConfig) SortCellsByPosition() {
	sort.SliceStable(c.Cells, func(i, j int) bool {
		pi := c.Cells[i].Position
		pj := c.Cells[j].Position
		if pi.Row != pj.Row {
			return pi.Row < pj.Row
		}
		return pi.Col < pj.Col
	})
}
