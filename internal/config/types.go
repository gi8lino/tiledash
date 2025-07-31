package config

import "time"

type DashboardConfig struct {
	Title           string        `yaml:"title"`
	Grid            Grid          `yaml:"grid"`
	Layout          []Section     `yaml:"layout"`
	RefreshInterval time.Duration `yaml:"refreshInterval"`
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
