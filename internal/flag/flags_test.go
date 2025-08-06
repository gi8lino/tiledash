package flag_test

import (
	"strings"
	"testing"

	"github.com/gi8lino/jirapanel/internal/flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGetEnv is only necessary since i use direnv which will interfear with my tests
func mockGetEnv(key string) string {
	return ""
}

func TestParseArgs(t *testing.T) {
	t.Parallel()

	t.Run("minimal", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://example.com/rest/api/3",
			"--jira-email=user@example.com",
			"--jira-auth=abc123",
		}
		var out strings.Builder

		cfg, err := flag.ParseArgs("v1.2.3", args, &out, mockGetEnv)
		require.NoError(t, err)
		require.Equal(t, "user@example.com", cfg.JiraEmail)
		require.Equal(t, "abc123", cfg.JiraAuth)
		require.Equal(t, "https://example.com/rest/api/3/", cfg.JiraAPIURL.String())
		require.Equal(t, "text", string(cfg.LogFormat))
		require.Equal(t, ":8080", cfg.ListenAddr)
	})

	t.Run("bearer-token", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://jira.org/rest/api/2",
			"--jira-bearer-token=bear123",
		}
		var out strings.Builder

		cfg, err := flag.ParseArgs("1.0.0", args, &out, mockGetEnv)
		require.NoError(t, err)
		require.Equal(t, "bear123", cfg.JiraBearerToken)
		require.Equal(t, "", cfg.JiraEmail)
		require.Equal(t, "", cfg.JiraAuth)
	})

	t.Run("invalid-email", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://site.com/rest/api/2",
			"--jira-email=invalid-email",
			"--jira-auth=t",
		}
		var out strings.Builder

		_, err := flag.ParseArgs("0.0.1", args, &out, mockGetEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "email must contain @")
	})

	t.Run("invalid jira api url", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://site.com/invalid/path",
			"--jira-bearer-token=bear",
		}
		var out strings.Builder

		_, err := flag.ParseArgs("0.0.1", args, &out, mockGetEnv)
		require.Error(t, err)
		require.Contains(t, err.Error(), "URL path must end with")
	})

	t.Run("override listen address", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://host.org/rest/api/2",
			"--jira-email=admin@host.org",
			"--jira-auth=abc",
			"--listen-address=127.0.0.1:9090",
		}
		var out strings.Builder

		cfg, err := flag.ParseArgs("1.0.0", args, &out, mockGetEnv)
		require.NoError(t, err)
		require.Equal(t, "127.0.0.1:9090", cfg.ListenAddr)
	})

	t.Run("json log format", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://jira/rest/api/3",
			"--jira-email=me@host.com",
			"--jira-auth=abc",
			"--log-format=json",
		}
		var out strings.Builder

		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv)
		require.NoError(t, err)
		require.Equal(t, "json", string(cfg.LogFormat))
	})

	t.Run("default templates dir", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://jira/rest/api/3",
			"--jira-email=test@jira.com",
			"--jira-auth=t",
		}
		var out strings.Builder

		cfg, err := flag.ParseArgs("dev", args, &out, mockGetEnv)
		require.NoError(t, err)
		require.Equal(t, "config.yaml", cfg.Config)
		require.Equal(t, "templates", cfg.TemplateDir)
	})

	t.Run("invalid jira timeout", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://jira/rest/api/3",
			"--jira-email=test@jira.com",
			"--jira-auth=t",
			"--jira-timeout=0s",
		}
		var out strings.Builder

		_, err := flag.ParseArgs("dev", args, &out, mockGetEnv)
		require.Error(t, err)
		assert.EqualError(t, err, "invalid value for flag --jira-timeout: timeout must be > 0.")
	})

	t.Run("valid jira timeout", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--jira-api-url=https://jira/rest/api/3",
			"--jira-email=test@jira.com",
			"--jira-auth=t",
			"--jira-timeout=1s",
		}
		var out strings.Builder

		_, err := flag.ParseArgs("dev", args, &out, mockGetEnv)
		require.NoError(t, err)
	})
}
