package config

import (
	"testing"
)

func TestGetLayoutByIndex(t *testing.T) {
	t.Parallel()

	t.Run("valid index", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Cells: []Cell{
				{Title: "A"},
				{Title: "B"},
			},
		}
		cell, err := cfg.GetLayoutByIndex(1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cell.Title != "B" {
			t.Errorf("expected Title 'B', got '%s'", cell.Title)
		}
	})

	t.Run("index too low", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Cells: []Cell{{Title: "Only"}},
		}
		_, err := cfg.GetLayoutByIndex(-1)
		if err == nil {
			t.Fatal("expected error for negative index, got nil")
		}
	})

	t.Run("index too high", func(t *testing.T) {
		t.Parallel()
		cfg := DashboardConfig{
			Cells: []Cell{{Title: "Only"}},
		}
		_, err := cfg.GetLayoutByIndex(2)
		if err == nil {
			t.Fatal("expected error for out-of-bounds index, got nil")
		}
	})
}
