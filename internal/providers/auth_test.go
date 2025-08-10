package providers

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gi8lino/tiledash/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestApplyAuth(t *testing.T) {
	t.Parallel()

	t.Run("nil auth -> no header", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
		applyAuth(req, nil)
		assert.Empty(t, req.Header.Get("Authorization"))
	})

	t.Run("basic auth", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
		a := &config.AuthConfig{Basic: &config.BasicAuth{Username: "u", Password: "p"}}
		applyAuth(req, a)
		h := req.Header.Get("Authorization")
		assert.True(t, strings.HasPrefix(h, "Basic "), "expected Basic auth header, got %q", h)
	})

	t.Run("bearer auth", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
		a := &config.AuthConfig{Bearer: &config.BearerAuth{Token: "tok"}}
		applyAuth(req, a)
		assert.Equal(t, "Bearer tok", req.Header.Get("Authorization"))
	})
}
