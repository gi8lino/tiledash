package flag_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gi8lino/jirapanel/internal/flag"
	"github.com/stretchr/testify/require"
)

func TestParseArgs_ValidMinimal(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://example.com/rest/api/3",
		"--jira-email=user@example.com",
		"--jira-auth=abc123",
	}
	var out strings.Builder

	cfg, err := flag.ParseArgs("v1.2.3", args, &out)
	require.NoError(t, err)
	require.Equal(t, "user@example.com", cfg.JiraEmail)
	require.Equal(t, "abc123", cfg.JiraAuth)
	require.Equal(t, "https://example.com/rest/api/3/", cfg.JiraAPIURL.String())
	require.Equal(t, "text", string(cfg.LogFormat))
	require.Equal(t, ":8080", cfg.ListenAddr)
}

func TestParseArgs_BearerToken(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://jira.org/rest/api/2",
		"--jira-bearer-token=bear123",
	}
	var out strings.Builder

	cfg, err := flag.ParseArgs("1.0.0", args, &out)
	require.NoError(t, err)
	require.Equal(t, "bear123", cfg.JiraBearerToken)
	require.Equal(t, "", cfg.JiraEmail)
	require.Equal(t, "", cfg.JiraAuth)
}

func TestParseArgs_InvalidEmail(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://site.com/rest/api/2",
		"--jira-email=invalid-email",
		"--jira-auth=t",
	}
	var out strings.Builder

	_, err := flag.ParseArgs("0.0.1", args, &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "email must contain @")
}

func TestParseArgs_InvalidJiraPath(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://site.com/invalid/path",
		"--jira-bearer-token=bear",
	}
	var out strings.Builder

	_, err := flag.ParseArgs("0.0.1", args, &out)
	require.Error(t, err)
	require.Contains(t, err.Error(), "URL path must end with")
}

func TestParseArgs_ListenAddressOverride(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://host.org/rest/api/2",
		"--jira-email=admin@host.org",
		"--jira-auth=abc",
		"--listen-address=127.0.0.1:9090",
	}
	var out strings.Builder

	cfg, err := flag.ParseArgs("1.0.0", args, &out)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:9090", cfg.ListenAddr)
}

func TestParseArgs_LogFormatJSON(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://jira/rest/api/3",
		"--jira-email=me@host.com",
		"--jira-auth=abc",
		"--log-format=json",
	}
	var out strings.Builder

	cfg, err := flag.ParseArgs("dev", args, &out)
	require.NoError(t, err)
	require.Equal(t, "json", string(cfg.LogFormat))
}

func TestParseArgs_TemplateConfigDefaults(t *testing.T) {
	t.Parallel()

	args := []string{
		"--jira-api-url=https://jira/rest/api/3",
		"--jira-email=test@jira.com",
		"--jira-auth=t",
	}
	var out strings.Builder

	cfg, err := flag.ParseArgs("dev", args, &out)
	require.NoError(t, err)
	require.Equal(t, "config.yaml", cfg.Config)
	require.Equal(t, "templates", cfg.TemplateDir)
}

func TestParseArgs_RefreshIntervalEnvFallback(t *testing.T) {
	t.Parallel()

	// Not settable via CLI directly in current implementation;
	// this test would be meaningful when env parsing is covered in integration tests.
	args := []string{
		"--jira-api-url=https://jira/rest/api/3",
		"--jira-email=test@jira.com",
		"--jira-auth=t",
	}
	var out strings.Builder

	cfg, err := flag.ParseArgs("dev", args, &out)
	require.NoError(t, err)
	require.Equal(t, time.Duration(0), cfg.RefreshInterval)
}
