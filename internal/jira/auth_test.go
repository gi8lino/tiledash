package jira_test

import (
	"net/http"
	"testing"

	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBasicAuth(t *testing.T) {
	t.Parallel()

	t.Run("sets basic auth header", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		auth := jira.NewBasicAuth(" user@example.com ", " token123 ")

		auth(req)

		username, password, ok := req.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "user@example.com", username)
		assert.Equal(t, "token123", password)
	})
}

func TestNewBearerAuth(t *testing.T) {
	t.Parallel()

	t.Run("sets bearer token header", func(t *testing.T) {
		t.Parallel()

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		auth := jira.NewBearerAuth("  abc123  ")

		auth(req)

		assert.Equal(t, "Bearer abc123", req.Header.Get("Authorization"))
	})
}

func TestResolveAuth(t *testing.T) {
	t.Parallel()

	t.Run("returns bearer auth when bearer token is provided", func(t *testing.T) {
		t.Parallel()

		auth, method, err := jira.ResolveAuth("mytoken", "", "")
		require.NoError(t, err)
		assert.Equal(t, "Bearer", method)

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		auth(req)
		assert.Equal(t, "Bearer mytoken", req.Header.Get("Authorization"))
	})

	t.Run("returns basic auth when email and token are provided", func(t *testing.T) {
		t.Parallel()

		auth, method, err := jira.ResolveAuth("", "me@example.com", "secret")
		require.NoError(t, err)
		assert.Equal(t, "Basic", method)

		req, _ := http.NewRequest("GET", "https://example.com", nil)
		auth(req)
		user, pass, ok := req.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "me@example.com", user)
		assert.Equal(t, "secret", pass)
	})

	t.Run("returns error when no credentials provided", func(t *testing.T) {
		t.Parallel()

		auth, method, err := jira.ResolveAuth("", "", "")
		assert.Error(t, err)
		assert.Nil(t, auth)
		assert.Empty(t, method)
	})
}
