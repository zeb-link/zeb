package cli

import (
	"strings"
	"testing"

	"github.com/kerns/zlink-zeb/internal/api"
)

func TestShortLink(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		path     string
		want     string
	}{
		{name: "plain path", hostname: "zbra.local", path: "abc", want: "zbra.local/abc"},
		{name: "leading slash", hostname: "zbra.local", path: "/abc", want: "zbra.local/abc"},
		{name: "empty path", hostname: "zbra.local", path: "", want: "zbra.local"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortLink(tt.hostname, tt.path); got != tt.want {
				t.Fatalf("shortLink(%q, %q) = %q, want %q", tt.hostname, tt.path, got, tt.want)
			}
		})
	}
}

func TestDisplayShortLinkPrefersServerShortURL(t *testing.T) {
	link := api.Link{
		Hostname: "zbra.local",
		Path:     "/abc",
		ShortURL: "https://zbra.local/abc",
	}

	if got := displayShortLink(link); got != link.ShortURL {
		t.Fatalf("displayShortLink() = %q, want %q", got, link.ShortURL)
	}
}

func TestDisplayShortLinkFallsBackToNormalizedHostnameAndPath(t *testing.T) {
	link := api.Link{
		Hostname: "zbra.local",
		Path:     "/abc",
	}

	if got := displayShortLink(link); got != "zbra.local/abc" {
		t.Fatalf("displayShortLink() = %q, want %q", got, "zbra.local/abc")
	}
}

func TestCreatedDomain(t *testing.T) {
	tests := []struct {
		name string
		link api.Link
		want string
	}{
		{
			name: "hostname from api response",
			link: api.Link{Hostname: "zbra.local:3001", ShortURL: "http://zbra.local:3001/abc"},
			want: "zbra.local:3001",
		},
		{
			name: "fallback to short url host",
			link: api.Link{ShortURL: "https://zlnk.to/abc"},
			want: "zlnk.to",
		},
		{
			name: "unknown when response has no domain fields",
			link: api.Link{},
			want: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createdDomain(tt.link); got != tt.want {
				t.Fatalf("createdDomain() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreatedCollection(t *testing.T) {
	if got := createdCollection("", ""); got != "none" {
		t.Fatalf("createdCollection() = %q, want none", got)
	}
	if got := createdCollection("col_123", "Launch"); got != "Launch" {
		t.Fatalf("createdCollection() = %q, want Launch", got)
	}
	if got := createdCollection("col_123", ""); got != "col_123" {
		t.Fatalf("createdCollection() = %q, want col_123", got)
	}
}

func TestReachabilityNote(t *testing.T) {
	if got := reachabilityNote(nil); got != "" {
		t.Fatalf("reachabilityNote(nil) = %q, want empty", got)
	}

	reachable := true
	if got := reachabilityNote(&reachable); !strings.Contains(got, "● verified") {
		t.Fatalf("reachabilityNote(true) = %q, want reachable dot", got)
	}

	reachable = false
	if got := reachabilityNote(&reachable); !strings.Contains(got, "● unreachable") {
		t.Fatalf("reachabilityNote(false) = %q, want unreachable dot", got)
	}
}
