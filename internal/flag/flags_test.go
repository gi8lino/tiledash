package flag_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gi8lino/tiledash/internal/flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGetEnv lets us simulate environment variables without touching the real env.
func mockGetEnv(vals map[string]string) func(string) string {
	return func(k string) string { return vals[k] }
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	t.Run("minimal defaults", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		cfg, err := flag.ParseArgs("v1.2.3", nil, &out, mockGetEnv(nil))
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

		var out strings.Builder
		cfg, err := flag.ParseArgs("v1.2.3", args, &out, mockGetEnv(nil))
		require.NoError(t, err)

		assert.Equal(t, "config.yaml", cfg.Config)
		assert.False(t, cfg.Debug)
		assert.Equal(t, "text", string(cfg.LogFormat))
		assert.Equal(t, ":8080", cfg.ListenAddr)

		assert.Equal(t, absPath, cfg.TemplateDir)
	})

	t.Run("site root", func(t *testing.T) {
		t.Parallel()

		args := []string{"--site-root=https://example.com"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("1.0.0", args, &out, mockGetEnv(nil))
		require.NoError(t, err)
		assert.Equal(t, "https://example.com", cfg.SiteRoot)
	})

	t.Run("listen address override", func(t *testing.T) {
		t.Parallel()

		args := []string{"--listen-address=127.0.0.1:9090"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("1.0.0", args, &out, mockGetEnv(nil))
		require.NoError(t, err)
		assert.Equal(t, "127.0.0.1:9090", cfg.ListenAddr)
	})

	t.Run("log format json", func(t *testing.T) {
		t.Parallel()

		args := []string{"--log-format=json"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.NoError(t, err)
		assert.Equal(t, "json", string(cfg.LogFormat))
	})

	t.Run("log format short flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"-l", "json"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.NoError(t, err)
		assert.Equal(t, "json", string(cfg.LogFormat))
	})

	t.Run("invalid log format", func(t *testing.T) {
		t.Parallel()

		args := []string{"--log-format=xml"} // not in {text,json}
		var out strings.Builder
		_, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.Error(t, err)
	})

	t.Run("debug flag", func(t *testing.T) {
		t.Parallel()

		args := []string{"--debug"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.NoError(t, err)
		assert.True(t, cfg.Debug)
	})

	t.Run("template dir relative is made absolute", func(t *testing.T) {
		t.Parallel()

		args := []string{"--template-dir=./testdata/templates"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
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
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.NoError(t, err)

		assert.Equal(t, abs, cfg.TemplateDir)
	})

	t.Run("config flag value", func(t *testing.T) {
		t.Parallel()

		args := []string{"--config=/path/to/my-config.yaml"}
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.NoError(t, err)
		assert.Equal(t, "/path/to/my-config.yaml", cfg.Config)
	})

	t.Run("from env variables", func(t *testing.T) {
		t.Parallel()

		env := map[string]string{
			"TILEDASH_LOG_FORMAT":   "json",
			"TILEDASH_DEBUG":        "true",
			"TILEDASH_CONFIG":       "env-config.yaml",
			"TILEDASH_TEMPLATE_DIR": "env-templates",
		}
		var out strings.Builder
		cfg, err := flag.ParseArgs("dev", nil, &out, mockGetEnv(env))
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
		var out strings.Builder
		_, err := flag.ParseArgs("dev", args, &out, mockGetEnv(nil))
		require.Error(t, err)
	})
}
