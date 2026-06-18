package app

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/gi8lino/tiledash/internal/flag"
	"github.com/gi8lino/tiledash/internal/logging"
	"github.com/gi8lino/tiledash/internal/providers"
	"github.com/gi8lino/tiledash/internal/routes"
	"github.com/gi8lino/tiledash/internal/templates"

	"github.com/containeroo/httpgrace/server"
	"github.com/containeroo/tinyflags"
)

// Run starts the tiledash application.
func Run(
	ctx context.Context,
	webFS fs.FS,
	version, commit string,
	args []string,
	stdOut, stdErr io.Writer,
) error {
	// Parse CLI flags
	flags, err := flag.ParseArgs(args, version)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			_, _ = fmt.Fprint(stdOut, err.Error())
			return nil
		}
		_, _ = fmt.Fprintln(stdErr, err)
		return err
	}

	// Setup logger immediately so startup errors are correctly logged.
	logger := logging.SetupLogger(flags, stdOut)
	setupLog := logger.With("component", "setup")
	setupLog.Info("Starting tiledash", "version", version)

	// Load config
	cfg, err := config.LoadConfig(flags.Config)
	if err != nil {
		setupLog.Error("Loading config error", "error", err)
		return err
	}

	// Try to parse user templates
	cellTmpl, err := templates.ParseCellTemplates(flags.TemplateDir, templates.TemplateFuncMap())
	if err != nil {
		setupLog.Error("template parsing error", "error", err)
		return err
	}

	// Validate config
	if err := cfg.Validate(cellTmpl); err != nil {
		setupLog.Error("config validation error", "error", err)
		return err
	}
	if flags.RoutePrefix != "" {
		setupLog.Debug("Using route prefix", "prefix", flags.RoutePrefix)
	}

	// Parse error template
	funcMap := templates.TemplateFuncMap()
	errTmpl := templates.ParseCellErrorTemplate(webFS, funcMap)

	if err := cfg.ResolveProvidersAuth(); err != nil {
		setupLog.Error("config provider auth resolution error", "error", err)
		return err
	}

	cfg.SortCellsByPosition() // Sorts all tiles top-to-bottom, left-to-right

	// Providers → registry
	reg, err := providers.BuildRegistry(cfg.Providers) // uses config.Provider
	if err != nil {
		setupLog.Error("error building registry", "error", err)
		return err
	}

	// Compile runners, one per tile
	runners, err := providers.BuildRunners(reg, cfg.Tiles)
	if err != nil {
		setupLog.Error("error building runners", "error", err)
		return err
	}

	// HTTP server
	serverLog := logger.With("component", "server")
	router := routes.NewRouter(
		webFS,
		cellTmpl,
		errTmpl,
		cfg,
		serverLog,
		runners,
		flags.Debug,
		version,
		flags.RoutePrefix,
	)

	ctx, stop := server.SignalContext(ctx)
	defer stop()

	if err := server.Run(ctx, flags.ListenAddr, router, serverLog); err != nil {
		setupLog.Error("server run", "listen_address", flags.ListenAddr, "error", err)
		return err
	}

	return nil
}
