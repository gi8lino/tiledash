package utils_test

import (
	"testing"

	"github.com/gi8lino/tiledash/internal/utils"
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
