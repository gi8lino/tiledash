package flag

import (
	"io"
	"net"
	"path/filepath"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/tiledash/internal/logging"
)

// Config holds all application and Jira-specific configuration.
type Config struct {
	ListenAddr  string            // HTTP bind address (e.g. ":8080")
	Debug       bool              // Enables debug logging
	LogFormat   logging.LogFormat // Log output format (text or json)
	Config      string            // Path to config file
	TemplateDir string            // Path to template directory
}

// ParseArgs parses CLI arguments into Config, handling version/help flags.
func ParseArgs(version string, args []string, out io.Writer, getEnv func(string) string) (Config, error) {
	var cfg Config
	tf := tinyflags.NewFlagSet("tiledash", tinyflags.ContinueOnError)
	tf.Version(version)
	tf.SetGetEnvFn(getEnv) // useful for testing
	tf.EnvPrefix("TILEDASH")
	tf.SetOutput(out)

	// Server
	tf.StringVar(&cfg.Config, "config", "config.yaml", "Path to config file").
		Value()

	tf.StringVar(&cfg.TemplateDir, "template-dir", "./templates", "Path to template directory").
		// Finalize only works if user has set a value
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
	tf.BoolVar(&cfg.Debug, "debug", false, "Enable debug logging").
		Value()
	logFormat := tf.String("log-format", "text", "Log format").
		Choices("text", "json").
		Short("l").
		Value()

	// Parse flags
	if err := tf.Parse(args); err != nil {
		return Config{}, err
	}
	// This needs to be done after parsing, since the flag value is not set until after the Parse call.
	cfg.LogFormat = logging.LogFormat(*logFormat)
	cfg.ListenAddr = (*listenAddr).String()
	if cfg.TemplateDir == "./templates" {
		// Finalize only works if user has set a value, not for defaults
		base := filepath.Dir(".")
		cfg.TemplateDir = filepath.Join(base, cfg.TemplateDir)
	}

	return cfg, nil
}
