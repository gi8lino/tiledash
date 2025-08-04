package app_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"github.com/gi8lino/jirapanel/internal/app"
	"github.com/gi8lino/jirapanel/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	t.Parallel()

	dummyEnv := func(key string) string { return "" }

	webFS := fstest.MapFS{
		"web/templates/base.gohtml":   &fstest.MapFile{Data: []byte(`{{define "base"}}{{end}}`)},
		"web/templates/footer.gohtml": &fstest.MapFile{Data: []byte(`{{define "footer"}}{{end}}`)},
		"web/templates/error.gohtml":  &fstest.MapFile{Data: []byte(`{{define "error"}}err{{end}}`)},
	}

	t.Run("Success", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		templateDir := filepath.Join(tmpDir, "templates")

		testutils.MustWriteFile(t, configPath, `
title: My Dashboard
grid:
  columns: 1
  rows: 1
layout:
  - title: Section 1
    query: filter=12345
    template: test.gohtml
    position: { row: 0, col: 0 }
refreshInterval: 30s
`)

		testutils.MustWriteFile(t, filepath.Join(templateDir, "test.gohtml"), `{{define "test"}}test{{end}}`)

		args := []string{
			"--config=" + configPath,
			"--template-dir=" + templateDir,
			"--jira-api-url=https://example.com/rest/api/2",
			"--jira-email=foo@example.com",
			"--jira-auth=xxx",
		}

		var buf bytes.Buffer

		err := app.Run(ctx, webFS, "v1", "cafe", args, &buf, func(string) string { return "" })
		require.NoError(t, err)
	})

	t.Run("Jira Auth Error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		args := []string{
			"--config=testdata/config.yaml",
			"--template-dir=testdata/templates",
			"--jira-api-url=https://example.com/rest/api/2",
			"--jira-email=foo@example.com",
			"--jira-auth=xxx",
			"--jira-bearer-token=zzz", // both -> error
		}

		var buf bytes.Buffer
		err := app.Run(ctx, webFS, "v1", "c0ffee", args, &buf, dummyEnv)

		require.Error(t, err)
		assert.EqualError(t, err, "parsing error: mutually exclusive flags used in group \"auth-method\": --jira-bearer-token vs [--jira-email, --jira-auth]")
	})

	t.Run("Help Requested", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := app.Run(ctx, webFS, "v1", "deadbeef", []string{"--help"}, &buf, dummyEnv)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "Usage")
	})

	t.Run("Version Requested", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := app.Run(ctx, webFS, "v1.2.3", "abc123", []string{"--version"}, &buf, dummyEnv)

		require.NoError(t, err)
		assert.Contains(t, buf.String(), "v1.2.3")
	})

	t.Run("Invalid Flag", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		var buf bytes.Buffer
		err := app.Run(ctx, webFS, "vX", "yyy", []string{"--unknown"}, &buf, dummyEnv)

		require.Error(t, err)
		assert.EqualError(t, err, "parsing error: unknown flag: --unknown")
	})

	t.Run("Config Error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		cfg := `
---
title: My Jira Dashboard
refreshInterval: 60s
grid:
  columns: 2
  rows: 4
layout:
  - title: Env Epics
    query: filter=17201
    template: epics.gohtml
    position: { row: 0, col: 0 }
`

		tmpdDir := t.TempDir()
		cfgPath := filepath.Join(tmpdDir, "config.yaml")
		testutils.MustWriteFile(t, cfgPath, cfg)

		args := []string{
			"--config=" + cfgPath,
			"--template-dir=testdata/templates",
			"--jira-api-url=https://example.com/rest/api/2",
			"--jira-email=foo@example.com",
			"--jira-auth=xxx",
		}

		var buf bytes.Buffer
		err := app.Run(ctx, webFS, "v1", "c0ffee", args, &buf, dummyEnv)

		require.Error(t, err)
		assert.EqualError(t, err, "validating config error: config validation failed:\n  - section[0] (Env Epics): template \"epics.gohtml\" not found")
	})

	t.Run("Config Validation Error", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
		defer cancel()

		cfg := `
---
title: My Jira Dashboard
refreshInterval: 60s
grid:
  columns: 2
  rows: 4
layout:
  - title: Env Epics
    query: filter=17201
    template: epics.gohtml
    position: { row: 0, col: 0 }
`

		cfgdDir := t.TempDir()
		cfgPath := filepath.Join(cfgdDir, "config.yaml")
		testutils.MustWriteFile(t, cfgPath, cfg)

		tmplDir := t.TempDir()
		testutils.MustWriteFile(t, filepath.Join(tmplDir, "bad.gohtml"), `{{define "bad"}}bad{{end}}`)

		args := []string{
			"--config=" + cfgPath,
			"--template-dir=" + tmplDir,
			"--jira-api-url=https://example.com/rest/api/2",
			"--jira-email=foo@example.com",
			"--jira-auth=xxx",
		}

		var buf bytes.Buffer
		err := app.Run(ctx, webFS, "v1", "c0ffee", args, &buf, dummyEnv)

		require.Error(t, err)
		assert.EqualError(t, err, "validating config error: config validation failed:\n  - section[0] (Env Epics): template \"epics.gohtml\" not found")
	})
}
