package config

import (
	"errors"
	"html/template"
	"time"
)

// DashboardConfig is the top-level configuration for the dashboard.
type DashboardConfig struct {
	Title           string              `yaml:"title"`
	RefreshInterval time.Duration       `yaml:"refreshInterval"`
	Grid            *GridConfig         `yaml:"grid"`
	Customization   *Customization      `yaml:"customization"`
	Providers       map[string]Provider `yaml:"providers"`
	Tiles           []Tile              `yaml:"tiles"`
}

// GridConfig controls the grid dimensions.
type GridConfig struct {
	Columns int `yaml:"columns"`
	Rows    int `yaml:"rows"`
}

// Customization collects optional presentation preferences.
type Customization struct {
	Grid   CustomGrid   `yaml:"grid"`
	Card   CustomCard   `yaml:"card"`
	Header CustomHeader `yaml:"header"`
	Footer CustomFooter `yaml:"footer"`
	Font   CustomFont   `yaml:"font"`
}

// CustomGrid customizes spacing around the grid.
type CustomGrid struct {
	Gap          template.CSS `yaml:"gap"`
	Padding      template.CSS `yaml:"padding"`
	MarginTop    template.CSS `yaml:"marginTop"`
	MarginBottom template.CSS `yaml:"marginBottom"`
}

// CustomCard customizes card visuals.
type CustomCard struct {
	BorderColor     template.CSS `yaml:"borderColor"`
	Padding         template.CSS `yaml:"padding"`
	BackgroundColor template.CSS `yaml:"backgroundColor"`
	BorderRadius    template.CSS `yaml:"borderRadius"`
	BoxShadow       template.CSS `yaml:"boxShadow"`
}

// CustomHeader customizes headers.
type CustomHeader struct {
	Align        template.CSS `yaml:"align"`
	MarginBottom template.CSS `yaml:"marginBottom"`
}

// CustomFooter customizes footers.
type CustomFooter struct {
	MarginTop template.CSS `yaml:"marginTop"`
}

// CustomFont customizes typography.
type CustomFont struct {
	Family template.CSS `yaml:"family"`
	Size   template.CSS `yaml:"size"`
}

// Provider defines a named upstream (baseURL + auth).
type Provider struct {
	BaseURL       string     `yaml:"baseURL"`
	SkipTLSVerify *bool      `yaml:"skipTLSVerify"`
	Auth          AuthConfig `yaml:"auth"`
}

// AuthConfig infers the scheme from which subfield is present.
type AuthConfig struct {
	Basic  *BasicAuth  `yaml:"basic,omitempty"`
	Bearer *BearerAuth `yaml:"bearer,omitempty"`
}

// BasicAuth carries username/password (or email/token) credentials.
type BasicAuth struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// BearerAuth carries a bearer token.
type BearerAuth struct {
	Token string `yaml:"token"`
}

// Tile is a single dashboard unit with layout + request.
type Tile struct {
	Title    string   `yaml:"title"`
	Template string   `yaml:"template"`
	Position Position `yaml:"position"`
	Request  Request  `yaml:"request"`
	Hash     string   `yaml:"-"` // computed hash
}

// Position places a tile in the grid.
type Position struct {
	Row     int `yaml:"row"`
	Col     int `yaml:"col"`
	ColSpan int `yaml:"colSpan"`
	RowSpan int `yaml:"rowSpan"`
}

// Request describes the HTTP request for a tile, bound to a provider.
type Request struct {
	Provider string            `yaml:"provider"`           // name under top-level providers
	Method   string            `yaml:"method,omitempty"`   // default GET
	Path     string            `yaml:"path"`               // relative to provider's BaseURL
	TTL      time.Duration     `yaml:"ttl,omitempty"`      // cache TTL
	Query    map[string]string `yaml:"query,omitempty"`    // query params
	Headers  map[string]string `yaml:"headers,omitempty"`  // extra headers
	Body     string            `yaml:"body,omitempty"`     // raw body
	BodyJSON map[string]any    `yaml:"bodyJSON,omitempty"` // JSON body (preferred)
	Paginate bool              `yaml:"paginate,omitempty"` // enable pagination
	Page     PageParams        `yaml:"page,omitempty"`     // pagination config
}

// PageParams configures offset/limit style pagination.
type PageParams struct {
	Location   string `yaml:"location,omitempty"` // "query" | "body"
	StartField string `yaml:"startField,omitempty"`
	LimitField string `yaml:"limitField,omitempty"`
	TotalField string `yaml:"totalField,omitempty"`
	ReqStart   string `yaml:"reqStart,omitempty"`
	ReqLimit   string `yaml:"reqLimit,omitempty"`
	LimitPages int    `yaml:"limitPages,omitempty"`
}

// GetCellByIndex returns a tile by index.
func (d DashboardConfig) GetCellByIndex(i int) (Tile, error) {
	if i < 0 || i >= len(d.Tiles) {
		return Tile{}, errors.New("index out of range")
	}
	return d.Tiles[i], nil
}
