package config

import (
	"testing"
)

func TestGetCellByIndex(t *testing.T) {
	t.Parallel()

	t.Run("valid index", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Tiles: []Tile{
				{Title: "A"},
				{Title: "B"},
			},
		}
		tile, err := cfg.GetCellByIndex(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tile.Title != "B" {
			t.Errorf("expected Title 'B', got '%s'", tile.Title)
		}
	})

	t.Run("index too low", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Tiles: []Tile{{Title: "Only"}},
		}
		_, err := cfg.GetCellByIndex(-1)
		if err == nil {
			t.Fatal("expected error for negative index, got nil")
		}
	})

	t.Run("index too high", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Tiles: []Tile{{Title: "Only"}},
		}
		_, err := cfg.GetCellByIndex(2)
		if err == nil {
			t.Fatal("expected error for out-of-bounds index, got nil")
		}
	})
}
