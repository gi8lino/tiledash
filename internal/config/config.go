package config

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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
func ValidateConfig(cfg DashboardConfig, tmpl *template.Template) error {
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

	occupied := make(map[[2]int]string) // track used grid cells

	for i, section := range cfg.Layout {
		label := fmt.Sprintf("section[%d] (%s)", i, section.Title)
		pos := section.Position

		if section.Title == "" {
			errs = append(errs, fmt.Sprintf("%s: title is required", label))
		}
		if section.Query == "" {
			errs = append(errs, fmt.Sprintf("%s: query is required", label))
		}
		if section.Template == "" {
			errs = append(errs, fmt.Sprintf("%s: template is required", label))
		} else if tmpl.Lookup(section.Template) == nil {
			errs = append(errs, fmt.Sprintf("%s: template %q not found", label, section.Template))
		}

		if pos.Row < 0 || pos.Row >= cfg.Grid.Rows {
			errs = append(errs, fmt.Sprintf("%s: row %d out of bounds (0–%d)", label, pos.Row, cfg.Grid.Rows-1))
		}
		if pos.Col < 0 || pos.Col >= cfg.Grid.Columns {
			errs = append(errs, fmt.Sprintf("%s: col %d out of bounds (0–%d)", label, pos.Col, cfg.Grid.Columns-1))
		}

		colSpan := pos.ColSpan
		if colSpan <= 0 {
			colSpan = 1
		}
		if pos.Col+colSpan > cfg.Grid.Columns {
			errs = append(errs, fmt.Sprintf("%s: colSpan %d overflows grid width %d", label, colSpan, cfg.Grid.Columns))
		}

		for c := pos.Col; c < pos.Col+colSpan; c++ {
			key := [2]int{pos.Row, c}
			if other, ok := occupied[key]; ok {
				errs = append(errs, fmt.Sprintf("%s: overlaps cell (%d,%d) used by section %q", label, pos.Row, c, other))
			}
			occupied[key] = section.Title
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}
