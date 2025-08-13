package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/flag"
	"github.com/gi8lino/tiledash/internal/logging"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/server"
	"github.com/gi8lino/tiledash/internal/templates"

	"github.com/containeroo/tinyflags"
)

// Run starts the tiledash application.
func Run(ctx context.Context, webFS fs.FS, version, commit string, args []string, w io.Writer, getEnv func(string) string) error {
	// Create a cantileable context on SIGINT/SIGTERM
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Parse CLI flags
	flags, err := flag.ParseArgs(version, args, w, getEnv)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(w, err.Error()) // nolint:errcheck
			return nil
		}
		return fmt.Errorf("parsing error: %w", err)
	}

	// Setup logger
	logger := logging.SetupLogger(flags.LogFormat, flags.Debug, w)
	logger.Info("Starting tiledash", "version", version)

	// Load config
	cfg, err := config.LoadConfig(flags.Config)
	if err != nil {
		return fmt.Errorf("loading config error: %w", err)
	}

	// Try to parse user templates
	tmpl, err := templates.ParseCellTemplates(flags.TemplateDir, templates.TemplateFuncMap())
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	// Validate config
	if err := cfg.Validate(tmpl); err != nil {
		return fmt.Errorf("config validation error: %w", err)
	}
	if flags.RoutePrefix != "" {
		logger.Debug("Using route prefix", "prefix", flags.RoutePrefix)
	}

	if err := cfg.ResolveProvidersAuth(); err != nil {
		return fmt.Errorf("config provider auth resolution error: %w", err)
	}

	cfg.SortCellsByPosition() // Sorts all tiles top-to-bottom, left-to-right

	// Providers → registry
	reg, err := providers.BuildRegistry(cfg.Providers) // uses config.Provider
	if err != nil {
		return fmt.Errorf("error building registry: %w", err)
	}

	// Compile runners, one per tile
	runners, err := providers.BuildRunners(reg, cfg.Tiles)
	if err != nil {
		return fmt.Errorf("error building runners: %w", err)
	}

	// HTTP server
	router := server.NewRouter(webFS, flags.TemplateDir, cfg, logger, runners, flags.Debug, version, flags.RoutePrefix)
	err = server.Run(ctx, flags.ListenAddr, router, logger)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("HTTP server exited with error", "error", err)
	}
	return err
}
