package testutils_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gi8lino/jirapanel/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestMockClient_SearchByJQL(t *testing.T) {
	t.Parallel()

	t.Run("forwards arguments and returns expected values", func(t *testing.T) {
		t.Parallel()

		expectedJQL := "project = TEST"
		expectedParams := map[string]string{"maxResults": "10"}
		expectedBody := []byte(`{"issues":[]}`)
		expectedStatus := 200

		// Capture the received values
		var gotJQL string
		var gotParams map[string]string

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				gotJQL = jql
				gotParams = params
				return expectedBody, expectedStatus, nil
			},
		}

		body, status, err := client.SearchByJQL(context.Background(), expectedJQL, expectedParams)

		require.NoError(t, err)
		assert.Equal(t, expectedStatus, status)
		assert.Equal(t, expectedBody, body)
		assert.Equal(t, expectedJQL, gotJQL)
		assert.Equal(t, expectedParams, gotParams)
	})

	t.Run("propagates errors from SearchFn", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("request failed")

		client := &testutils.MockClient{
			SearchFn: func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
				return nil, 500, expectedErr
			},
		}

		body, status, err := client.SearchByJQL(t.Context(), "any", nil)

		require.Error(t, err)
		assert.Equal(t, 500, status)
		assert.Nil(t, body)
		assert.Equal(t, expectedErr, err)
	})
}
