package providers

import (
	"testing"
)

// TestMergeCommonArrays_Dedup ensures overlapping pages don't produce duplicates.
func TestMergeCommonArrays_Dedup(t *testing.T) {
	t.Parallel()

	acc := newAccumulator()

	// page1: ids 1,2
	page1 := map[string]any{
		"issues": []any{
			map[string]any{"id": 1, "key": "K-1"},
			map[string]any{"id": 2, "key": "K-2"},
		},
	}
	mergeCommonArrays(acc, page1)

	// page2 overlaps: ids 2,3
	page2 := map[string]any{
		"issues": []any{
			map[string]any{"id": 2, "key": "K-2"},
			map[string]any{"id": 3, "key": "K-3"},
		},
	}
	mergeCommonArrays(acc, page2)

	merged, _ := acc["merged"].(map[string]any)
	iss, _ := merged["issues"].([]any)
	if got, want := len(iss), 3; got != want {
		t.Fatalf("merged issues length=%d, want %d", got, want)
	}
}
