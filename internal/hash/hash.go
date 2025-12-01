package hash

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
)

// Any serializes the given value and returns its FNV-1a 64-bit hash as a hex string.
func Any(a any) (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("failed to serialize: %w", err)
	}

	h := fnv.New64a()
	if _, err := h.Write(data); err != nil {
		return "", fmt.Errorf("failed to hash data: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum64()), nil
}
