package flag

import (
	"io"
	"net"
	"path/filepath"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/tiledash/internal/logging"
	"github.com/gi8lino/tiledash/internal/utils"
)

// Config holds all application and Jira-specific configuration.
type Config struct { // Config aggregates CLI flags after parsing.
	ListenAddr  string            // HTTP bind address (e.g. ":8080")
	Debug       bool              // Enables debug logging
	LogFormat   logging.LogFormat // Log output format (text or json)
	Config      string            // Path to config file
	TemplateDir string            // Path to template directory
	RoutePrefix string            // Canonical path prefix ("" or "/tiledash")
}

// ParseArgs parses CLI arguments into Config, handling version/help flags.
func ParseArgs(version string, args []string, out io.Writer, getEnv func(string) string) (Config, error) { // ParseArgs parses CLI args into Config.
	var cfg Config
	tf := tinyflags.NewFlagSet("tiledash", tinyflags.ContinueOnError)
	tf.Version(version)
	tf.SetGetEnvFn(getEnv)
	tf.EnvPrefix("TILEDASH")
	tf.SetOutput(out)

	// Server
	tf.StringVar(&cfg.Config, "config", "config.yaml", "Path to config file").Value()

	route := tf.String("route-prefix", "", "Path prefix to mount the app (e.g., /tiledash). Empty = root.").
		Finalize(func(input string) string {
			return utils.NormalizeRoutePrefix(input) // canonical "" or "/tiledash"
		}).
		Placeholder("PATH").
		Value()

	tf.StringVar(&cfg.TemplateDir, "template-dir", "./templates", "Path to template directory").
		Finalize(func(s string) string {
			if !filepath.IsAbs(s) {
				base := filepath.Dir(".")
				path, _ := filepath.Abs(filepath.Join(base, s))
				return path
			}
			return s
		}).
		Value()

	listenAddr := tf.TCPAddr("listen-address", &net.TCPAddr{IP: nil, Port: 8080}, "HTTP server listen address").
		Placeholder("ADDR:PORT").
		Value()

	// Logging
	tf.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging").Value()
	logFormat := tf.String("log-format", "text", "Log format").Choices("text", "json").Short("l").Value()

	// Parse
	if err := tf.Parse(args); err != nil {
		return Config{}, err
	}

	// Post-parse
	cfg.LogFormat = logging.LogFormat(*logFormat)
	cfg.ListenAddr = (*listenAddr).String()
	cfg.RoutePrefix = *route

	if cfg.TemplateDir == "./templates" {
		base := filepath.Dir(".")
		cfg.TemplateDir = filepath.Join(base, cfg.TemplateDir)
	}

	return cfg, nil
}
