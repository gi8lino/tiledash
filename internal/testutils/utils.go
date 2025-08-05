package testutils

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// MustWriteFile writes data to a file or fails the test, creating parent directories if needed.
func MustWriteFile(t *testing.T, path, content string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create directory %q: %v", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write test file %q: %v", path, err)
	}
}

// mockClient is a mock implementation of the jira.Searcher interface.
type MockClient struct {
	SearchFn func(ctx context.Context, jql string, params map[string]string) ([]byte, int, error)
}

// SearchByJQL implements the jira.Searcher interface.
func (m *MockClient) SearchByJQL(ctx context.Context, jql string, params map[string]string) ([]byte, int, error) {
	return m.SearchFn(ctx, jql, params)
}
