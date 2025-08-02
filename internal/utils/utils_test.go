package utils_test

import (
	"strings"
	"testing"

	"github.com/gi8lino/jirapanel/internal/jira"
	"github.com/gi8lino/jirapanel/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestObfuscateHeader(t *testing.T) {
	t.Parallel()

	t.Run("returns empty on empty input", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "", utils.ObfuscateHeader(""))
	})

	t.Run("returns invalid if no scheme", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "[invalid header]", utils.ObfuscateHeader("invalidheader"))
	})

	t.Run("obfuscates token with full length > 4", func(t *testing.T) {
		t.Parallel()
		result := utils.ObfuscateHeader("Bearer abcdefghijkl")
		assert.Equal(t, "Bearer ab********kl", result)
	})

	t.Run("obfuscates short token length <= 4", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Bearer ****", utils.ObfuscateHeader("Bearer abcd"))
		assert.Equal(t, "Bearer ***", utils.ObfuscateHeader("Bearer abc"))
		assert.Equal(t, "Bearer **", utils.ObfuscateHeader("Bearer ab"))
		assert.Equal(t, "Bearer *", utils.ObfuscateHeader("Bearer a"))
		assert.Equal(t, "Bearer ", utils.ObfuscateHeader("Bearer "))
	})
}

func TestGetAuthorizationHeader(t *testing.T) {
	t.Parallel()

	t.Run("gets basic auth header", func(t *testing.T) {
		t.Parallel()
		authFunc := jira.NewBasicAuth("user@example.com", "secret123")
		header := utils.GetAuthorizationHeader(authFunc)
		assert.True(t, strings.HasPrefix(header, "Basic "))
		assert.Equal(t, "Basic dXNlckBleGFtcGxlLmNvbTpzZWNyZXQxMjM=", header)
	})

	t.Run("gets bearer token header", func(t *testing.T) {
		t.Parallel()
		authFunc := jira.NewBearerAuth("my-token-xyz")
		header := utils.GetAuthorizationHeader(authFunc)
		assert.Equal(t, "Bearer my-token-xyz", header)
	})
}
