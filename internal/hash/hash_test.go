package hash

import (
	"testing"
)

func TestAnyDeterministic(t *testing.T) {
	t.Parallel()

	input := map[string]any{"foo": "bar", "baz": 1}

	first, err := Any(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := Any(input)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic hash, got %q and %q", first, second)
	}
}

func TestAnyNonSerializable(t *testing.T) {
	t.Parallel()

	ch := make(chan int)
	if _, err := Any(ch); err == nil {
		t.Fatalf("expected error for non-serializable input")
	}
}
