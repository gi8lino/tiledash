// Package config defines the structure for the dashboard configuration,
// including layout, cell definitions, styling, and refresh settings.
package config

import (
	"fmt"
	"time"
)

// DashboardConfig holds the full configuration for the rendered dashboard.
type DashboardConfig struct {
	Title           string         `yaml:"title"`                   // Title is the main title shown at the top of the dashboard.
	Grid            Grid           `yaml:"grid"`                    // Grid defines the number of rows and columns in the dashboard layout.
	Cells           []Cell         `yaml:"cells"`                   // Cells are the individual dashboard cells to be rendered inside the grid.
	RefreshInterval time.Duration  `yaml:"refreshInterval"`         // RefreshInterval defines how frequently the dashboard auto-refreshes.
	Customization   *Customization `yaml:"customization,omitempty"` // Customization holds styling overrides for layout, font, spacing, etc.
}

// GetLayoutByIndex returns the Cell at the given index or an error if out of range.
func (c *DashboardConfig) GetLayoutByIndex(idx int) (*Cell, error) {
	if idx < 0 || idx >= len(c.Cells) {
		return nil, fmt.Errorf("cell index %d out of bounds", idx)
	}
	return &c.Cells[idx], nil
}

// Grid defines the number of columns and rows used in the dashboard grid.
type Grid struct {
	Columns int `yaml:"columns"` // Columns is the number of columns in the grid.
	Rows    int `yaml:"rows"`    // Rows is the number of rows in the grid.
}

// Cell represents a single unit inside the dashboard grid.
type Cell struct {
	Hash     string            `yaml:"-"`        // Hash is the hash of the cell content.
	Title    string            `yaml:"title"`    // Title is the visible title shown above the cell content.
	Query    string            `yaml:"query"`    // Query is the JQL query used to fetch issues for this cell.
	Params   map[string]string `yaml:"params"`   // Params is an optional map of extra query parameters (e.g. maxResults).
	Template string            `yaml:"template"` // Template is the name of the template used to render the cell content.
	Position Position          `yaml:"position"` // Position determines where this cell is placed in the dashboard grid.
}

// Position specifies the grid position and span for a given cell.
type Position struct {
	Row     int `yaml:"row"`               // Row is the zero-based row index of the cell.
	Col     int `yaml:"col"`               // Col is the zero-based column index of the cell.
	ColSpan int `yaml:"colSpan,omitempty"` // ColSpan optionally allows the cell to span multiple columns.
}

// Customization groups all styling-related fields for the dashboard.
type Customization struct {
	Grid   GridStyle   `yaml:"grid"`   // Grid defines padding, gap, and margins for the grid container.
	Card   CardStyle   `yaml:"card"`   // Card controls the style of each individual cell/card.
	Header HeaderStyle `yaml:"header"` // Header customizes the main title header.
	Footer FooterStyle `yaml:"footer"` // Footer customizes spacing below the grid.
	Font   FontStyle   `yaml:"font"`   // Font sets global font family and size.
}

// GridStyle controls spacing and margins within the dashboard grid.
type GridStyle struct {
	Gap          string `yaml:"gap"`          // Gap defines spacing between grid items (e.g. "1rem").
	Padding      string `yaml:"padding"`      // Padding is the internal padding of the grid container.
	MarginTop    string `yaml:"marginTop"`    // MarginTop sets the top margin of the grid section.
	MarginBottom string `yaml:"marginBottom"` // MarginBottom sets the bottom margin of the grid section.
}

// CardStyle controls the appearance of each dashboard cell/card.
type CardStyle struct {
	BorderColor     string `yaml:"borderColor"`     // BorderColor is the CSS border color of the card.
	Padding         string `yaml:"padding"`         // Padding is the internal padding inside the card.
	BackgroundColor string `yaml:"backgroundColor"` // BackgroundColor is the fill/background color of the card.
	BorderRadius    string `yaml:"borderRadius"`    // BorderRadius defines how rounded the card edges are.
	BoxShadow       string `yaml:"boxShadow"`       // BoxShadow sets the shadow used behind the card.
}

// HeaderStyle customizes the appearance of the main dashboard title.
type HeaderStyle struct {
	Align        string `yaml:"align"`        // Align is the text alignment of the title (e.g. "center").
	MarginBottom string `yaml:"marginBottom"` // MarginBottom adds spacing below the title.
}

// FooterStyle defines the spacing below the dashboard.
type FooterStyle struct {
	MarginTop string `yaml:"marginTop"` // MarginTop adds spacing between the grid and the footer.
}

// FontStyle configures the global font used across the dashboard.
type FontStyle struct {
	Family string `yaml:"family"` // Family is the CSS font-family string (e.g. "monospace").
	Size   string `yaml:"size"`   // Size is the base font size (e.g. "14px", "1rem").
}
