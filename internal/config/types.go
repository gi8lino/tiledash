package config

import "time"

type DashboardConfig struct {
	Title           string        `yaml:"title"`
	Grid            Grid          `yaml:"grid"`
	Layout          []Section     `yaml:"layout"`
	RefreshInterval time.Duration `yaml:"refreshInterval"`
	Customization   Customization `yaml:"customization"`
}

type Grid struct {
	Columns int `yaml:"columns"`
	Rows    int `yaml:"rows"`
}

type Section struct {
	Title    string            `yaml:"title"`
	Query    string            `yaml:"query"`
	Params   map[string]string `yaml:"params"` // optional extras (e.g. maxResults)
	Template string            `yaml:"template"`
	Position Position          `yaml:"position"`
}

type Position struct {
	Row     int `yaml:"row"`
	Col     int `yaml:"col"`
	ColSpan int `yaml:"colSpan,omitempty"`
}

type Customization struct {
	Grid   GridStyle   `yaml:"grid"`
	Card   CardStyle   `yaml:"card"`
	Header HeaderStyle `yaml:"header"`
	Font   FontStyle   `yaml:"font"`
}

type GridStyle struct {
	Gap       string `yaml:"gap"`
	Padding   string `yaml:"padding"`
	MarginTop string `yaml:"marginTop"`
}

type CardStyle struct {
	BorderColor     string `yaml:"borderColor"`
	Padding         string `yaml:"padding"`
	BackgroundColor string `yaml:"backgroundColor"`
	BorderRadius    string `yaml:"borderRadius"`
	BoxShadow       string `yaml:"boxShadow"`
}

type HeaderStyle struct {
	Align        string `yaml:"align"`
	MarginBottom string `yaml:"marginBottom"`
}

type FontStyle struct {
	Family string `yaml:"family"`
	Size   string `yaml:"size"`
}
