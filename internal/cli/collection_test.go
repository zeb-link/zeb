package cli

import (
	"strings"
	"testing"

	"github.com/kerns/zlink-zeb/internal/api"
)

func TestCollectionLinkCountLabel(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{count: 0, want: "0 links"},
		{count: 1, want: "1 link"},
		{count: 2, want: "2 links"},
	}

	for _, tt := range tests {
		if got := collectionLinkCountLabel(tt.count); got != tt.want {
			t.Fatalf("collectionLinkCountLabel(%d) = %q, want %q", tt.count, got, tt.want)
		}
	}
}

func TestCollectionDescription(t *testing.T) {
	if got := collectionDescription(api.Collection{}); got != "" {
		t.Fatalf("collectionDescription(nil) = %q, want empty", got)
	}

	description := "  Saved reading list  "
	if got := collectionDescription(api.Collection{Description: &description}); got != "Saved reading list" {
		t.Fatalf("collectionDescription() = %q, want trimmed description", got)
	}
}

func TestCollectionStatus(t *testing.T) {
	_, active := collectionStatus(api.Collection{Type: "manual"}, true)
	if !strings.Contains(active, "active") {
		t.Fatalf("collectionStatus(active) = %q, want active", active)
	}

	_, smart := collectionStatus(api.Collection{Type: "smart"}, false)
	if smart != "" {
		t.Fatalf("collectionStatus(smart) = %q, want empty row label", smart)
	}

	_, manual := collectionStatus(api.Collection{Type: "manual"}, false)
	if manual != "" {
		t.Fatalf("collectionStatus(manual) = %q, want empty row label", manual)
	}
}

func TestCollectionHeadingIncludesLegend(t *testing.T) {
	heading := collectionHeading()
	for _, text := range []string{"Collections", "collection", "smart", "active"} {
		if !strings.Contains(heading, text) {
			t.Fatalf("collectionHeading() = %q, want %q", heading, text)
		}
	}
}

func TestCollectionCommandHasLinksSubcommand(t *testing.T) {
	cmd := newCollectionCommand(&rootOptions{})
	found, _, err := cmd.Find([]string{"links"})
	if err != nil {
		t.Fatalf("Find(links) returned error: %v", err)
	}
	if found == nil || found.Name() != "links" {
		t.Fatalf("collection links command not found")
	}
}
