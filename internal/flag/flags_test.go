package flag_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gi8lino/tiledash/internal/flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	t.Run("minimal defaults", func(t *testing.T) {
		t.Parallel()

		cfg, err := flag.ParseArgs(nil, "v1.2.3")
		require.NoError(t, err)

		assert.Equal(t, "config.yaml", cfg.Config)
		assert.False(t, cfg.Debug)
		assert.Equal(t, "text", string(cfg.LogFormat))
		assert.Equal(t, ":8080", cfg.ListenAddr)

		// TemplateDir is finalized to an absolute path ending with "templates".
		assert.Equal(t, "templates", cfg.TemplateDir)
	})

	t.Run("template dir relative is made absolute", func(t *testing.T) {
		t.Parallel()

		absPath, _ := filepath.Abs("./testdata/templates")
		args := []string{"--template-dir=./testdata/templates"}

		cfg, err := flag.ParseArgs(args, "v1.2.3")
		require.NoError(t, err)

		assert.Equal(t, "config.yaml", cfg.Config)
		assert.False(t, cfg.Debug)
		assert.Equal(t, "text", string(cfg.LogFormat))
		assert.Equal(t, ":8080", cfg.ListenAddr)
		assert.Equal(t, absPath, cfg.TemplateDir)
	})

	t.Run("route prefix default empty", func(t *testing.T) {
		t.Parallel()

		cfg, err := flag.ParseArgs([]string{}, "v")
		require.NoError(t, err)
		assert.Empty(t, cfg.RoutePrefix)
	})

	t.Run("route-prefix normalized value", func(t *testing.T) {
		t.Parallel()

		cfg, err := flag.ParseArgs([]string{"--route-prefix=tiledash/"}, "v")
		require.NoError(t, err)
		assert.Equal(t, cfg.RoutePrefix, "/tiledash")
	})

	t.Run("listen address override", func(t *testing.T) {
		t.Parallel()

		args := []string{"--listen-address=127.0.0.1:9090"}
		cfg, err := flag.ParseArgs(args, "1.0.0")
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1:9090", cfg.ListenAddr)
	})

	t.Run("log format json", func(t *testing.T) {
		t.Parallel()

		args := []string{"--log-format=json"}
		cfg, err := flag.ParseArgs(args, "dev")
		require.NoError(t, err)
		assert.Equal(t, "json", string(cfg.LogFormat))
	})

	t.Run("log format short flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"-l", "json"}
		cfg, err := flag.ParseArgs(args, "dev")
		require.NoError(t, err)
		assert.Equal(t, "json", string(cfg.LogFormat))
	})

	t.Run("invalid log format", func(t *testing.T) {
		t.Parallel()

		args := []string{"--log-format=xml"} // not in {text,json}
		_, err := flag.ParseArgs(args, "dev")
		require.Error(t, err)
	})

	t.Run("debug flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"--debug"}
		cfg, err := flag.ParseArgs(args, "dev")
		require.NoError(t, err)
		assert.True(t, cfg.Debug)
	})

	t.Run("template dir relative is made absolute", func(t *testing.T) {
		t.Parallel()

		args := []string{"--template-dir=./testdata/templates"}
		cfg, err := flag.ParseArgs(args, "dev")
		require.NoError(t, err)

		assert.True(t, filepath.IsAbs(cfg.TemplateDir))
		assert.True(t, strings.HasSuffix(
			filepath.ToSlash(cfg.TemplateDir),
			"/testdata/templates",
		))
	})

	t.Run("template dir absolute stays absolute", func(t *testing.T) {
		t.Parallel()

		abs := filepath.Join(t.TempDir(), "tpls")
		args := []string{"--template-dir=" + abs}
		cfg, err := flag.ParseArgs(args, "dev")
		require.NoError(t, err)

		assert.Equal(t, abs, cfg.TemplateDir)
	})

	t.Run("config flag value", func(t *testing.T) {
		t.Parallel()

		args := []string{"--config=/path/to/my-config.yaml"}
		cfg, err := flag.ParseArgs(args, "dev")
		require.NoError(t, err)
		assert.Equal(t, "/path/to/my-config.yaml", cfg.Config)
	})

	t.Run("from env variables", func(t *testing.T) {
		t.Setenv("TILEDASH_LOG_FORMAT", "json")
		t.Setenv("TILEDASH_DEBUG", "true")
		t.Setenv("TILEDASH_CONFIG", "env-config.yaml")
		t.Setenv("TILEDASH_TEMPLATE_DIR", "env-templates")

		cfg, err := flag.ParseArgs(nil, "dev")
		require.NoError(t, err)

		assert.Equal(t, "json", string(cfg.LogFormat))
		assert.True(t, cfg.Debug)
		assert.Equal(t, "env-config.yaml", cfg.Config)

		// template-dir from env should also be finalized to absolute
		assert.True(t, filepath.IsAbs(cfg.TemplateDir))
		assert.True(t, strings.HasSuffix(filepath.ToSlash(cfg.TemplateDir), "/env-templates"))
	})

	t.Run("invalid listen address", func(t *testing.T) {
		t.Parallel()

		args := []string{"--listen-address=not-an-addr"}
		_, err := flag.ParseArgs(args, "dev")
		require.Error(t, err)
	})
}
