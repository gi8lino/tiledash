package flag

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/jirapanel/internal/logging"
)

// Config holds all application and Jira-specific configuration.
type Config struct {
	ListenAddr        string            // HTTP bind address (e.g. ":8080")
	APIToken          string            // Shared token to authorize external API calls
	JiraAPIURL        *url.URL          // Parsed Jira REST base URL (must end with /rest/api/2 or /3)
	JiraEmail         string            // Email for cloud/basic authentication
	JiraAuth          string            // API token or password
	JiraBearerToken   string            // Bearer token for self-hosted Jira
	JiraSkipTLSVerify bool              // If true, disables TLS verification
	Debug             bool              // Enables debug logging
	LogFormat         logging.LogFormat // Log output format (text or json)
	Config            string            // Path to config file
	TemplateDir       string            // Path to template directory
	RefreshInterval   time.Duration     // Refresh interval for dashboard
}

// ParseArgs parses CLI arguments into Config, handling version/help flags.
func ParseArgs(version string, args []string, out io.Writer, getEnv func(string) string) (Config, error) {
	var cfg Config
	tf := tinyflags.NewFlagSet("jirapanel", tinyflags.ContinueOnError)
	tf.Version(version)
	tf.SetGetEnvFn(getEnv) // useful for testing
	tf.EnvPrefix("JIRAPANEL")
	tf.SetOutput(out)

	// Server
	tf.StringVar(&cfg.Config, "config", "config.yaml", "Path to config file").
		Value()
	tf.StringVar(&cfg.TemplateDir, "template-dir", "templates", "Path to template directory. ").
		Value()
	listenAddr := tf.TCPAddr("listen-address", &net.TCPAddr{IP: nil, Port: 8080}, "HTTP server listen address").
		Placeholder("ADDR:PORT").
		Value()

	// Jira connection
	tf.URLVar(&cfg.JiraAPIURL, "jira-api-url", &url.URL{}, "Base Jira REST API URL (e.g. https://org.atlassian.net/rest/api/3)").
		Finalize(func(u *url.URL) *url.URL {
			// Clone to avoid mutating the original (optional, if needed)
			u2 := *u
			if len(u2.Path) > 0 && u2.Path[len(u2.Path)-1] != '/' {
				u2.Path += "/"
			}
			return &u2
		}).
		Validate(func(u *url.URL) error {
			switch u.Path {
			case "/rest/api/2/", "/rest/api/3/", "/rest/api/2", "/rest/api/3":
				return nil
			default:
				return fmt.Errorf("URL path must end with /rest/api/2 or /rest/api/3, got %q", u.Path)
			}
		}).
		Placeholder("URL").
		Value()

	// Jira authentication
	tf.StringVar(&cfg.JiraEmail, "jira-email", "", "Email for cloud/basic authentication").
		Validate(func(email string) error {
			if !strings.Contains(email, "@") {
				return fmt.Errorf("email must contain @")
			}
			return nil
		}).
		AllOrNone("basic-auth").
		Placeholder("EMAIL").
		Value()
	tf.StringVar(&cfg.JiraAuth, "jira-auth", "", "Password or API token (used with --jira-email)").
		Placeholder("TOKEN").
		AllOrNone("basic-auth").
		Value()
	tf.GetOneOfGroup("auth-method").
		AddGroup(tf.GetAllOrNoneGroup("basic-auth"))

	tf.StringVar(&cfg.JiraBearerToken, "jira-bearer-token", "", "Bearer token for self-hosted Jira").
		OneOfGroup("auth-method").
		Placeholder("BEARER").
		Value()

	tf.BoolVar(&cfg.JiraSkipTLSVerify, "jira-skip-tls-verify", false, "Disable TLS verification for Jira connections").
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

	return cfg, nil
}
