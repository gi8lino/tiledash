package testutils_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gi8lino/jirapanel/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// TestMustWriteFile ensures that MustWriteFile creates files and parent directories correctly.
func TestMustWriteFile(t *testing.T) {
	t.Parallel()

	t.Run("creates file with content", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "subdir", "testfile.txt")
		expected := "hello, world"

		testutils.MustWriteFile(t, filePath, expected)

		data, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, expected, string(data))
	})
}
