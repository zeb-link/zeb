package layout

import "testing"

func TestTruncateCountsRunesNotBytes(t *testing.T) {
	tests := []struct {
		name  string
		value string
		limit int
		want  string
	}{
		{"short unchanged", "hello", 10, "hello"},
		{"exact limit unchanged", "hello", 5, "hello"},
		{"ascii truncated", "hello world", 6, "hello…"},
		{"limit one returns value", "hello", 1, "hello"},
		// The whole point: a multibyte string must be sliced on rune
		// boundaries, never bytes — byte slicing would split "★" or "中".
		{"multibyte not split", "café★中文字", 4, "caf…"},
		{"multibyte kept when short", "中文", 5, "中文"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.value, tt.limit); got != tt.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", tt.value, tt.limit, got, tt.want)
			}
		})
	}
}

func TestGridComputesColumnsFromWidth(t *testing.T) {
	tiles := []string{"a", "b", "c", "d"}

	// tileWidth 1, gap 1 → each column costs 2 cells. Width 3 fits 2 columns.
	if got, want := Grid(tiles, 1, 1, 3), "a b\nc d"; got != want {
		t.Fatalf("Grid width 3 = %q, want %q", got, want)
	}

	// Width 1 can only fit a single column.
	if got, want := Grid(tiles, 1, 1, 1), "a\nb\nc\nd"; got != want {
		t.Fatalf("Grid width 1 = %q, want %q", got, want)
	}

	// Ample width fits every tile on one row.
	if got, want := Grid(tiles, 1, 1, 100), "a b c d"; got != want {
		t.Fatalf("Grid width 100 = %q, want %q", got, want)
	}

	if got := Grid(nil, 1, 1, 10); got != "" {
		t.Fatalf("Grid(nil) = %q, want empty", got)
	}
}
