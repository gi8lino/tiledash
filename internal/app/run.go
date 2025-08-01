package app

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"

	"github.com/gi8lino/jirapanel/internal/config"
	"github.com/gi8lino/jirapanel/internal/flag"
	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/logging"
	"github.com/gi8lino/jirapanel/internal/server"
	"github.com/gi8lino/jirapanel/internal/templates"
	"github.com/gi8lino/jirapanel/internal/utils"

	"github.com/containeroo/tinyflags"
)

// Run starts the package-exporter application.
func Run(ctx context.Context, webFS fs.FS, version, commit string, args []string, w io.Writer) error {
	// Create a new context that listens for interrupt signals
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Parse command-line flags
	flags, err := flag.ParseArgs(version, args, w)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(w, err.Error()) // nolint:errcheck
			return nil
		}
		return fmt.Errorf("parsing error: %w", err)
	}

	// Setup logger
	logger := logging.SetupLogger(flags.LogFormat, flags.Debug, w)

	logger.Info("Starting jirapanel",
		"version", version,
	)

	// Load config
	cfg, err := config.LoadConfig(flags.Config)
	if err != nil {
		return fmt.Errorf("loading config error: %w", err)
	}

	// Try to load templates
	tmpl, err := template.New("").
		Funcs(templates.TemplateFuncMap()).
		ParseGlob(path.Join(flags.TemplateDir, "*.gohtml"))
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	// Validate config
	if err := config.ValidateConfig(cfg, tmpl); err != nil {
		return fmt.Errorf("validating config error: %w", err)
	}

	// Setup jira client
	var auth jira.AuthFunc
	var method string
	switch {
	case flags.JiraBearerToken != "":
		auth = jira.NewBearerAuth(flags.JiraBearerToken)
		method = "Bearer"
	case flags.JiraEmail != "" && flags.JiraAuth != "":
		auth = jira.NewBasicAuth(flags.JiraEmail, flags.JiraAuth)
		method = "Basic"
	default:
		return fmt.Errorf("no valid auth method configured")
	}
	c := jira.NewClient(flags.JiraAPIURL, auth, flags.JiraSkipTLSVerify)

	logger.Debug("jira auth",
		"method", method,
		"header", utils.ObfuscateHeader(utils.GetAuthorizationHeader(auth)),
	)

	// Setup Server and run forever
	router := server.NewRouter(
		webFS,
		flags.TemplateDir,
		c,
		cfg,
		logger,
		flags.Debug,
		version,
	)
	err = server.RunHTTPServer(ctx, router, flags.ListenAddr, logger)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server exited with error", "error", err)
	}

	return err
}
