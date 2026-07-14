package main

import "testing"

func TestNormalizeModuleVersion(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"released tag", "v0.1.0", "0.1.0"},
		{"prerelease tag", "v1.2.0-rc.1", "1.2.0-rc.1"},
		{"untagged commit", "v0.0.0-20260714185951-1cb3d7ca9d5b", "0.0.0-20260714185951-1cb3d7ca9d5b"},
		{"working tree build", "(devel)", devVersion},
		{"no build info", "", devVersion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeModuleVersion(tc.in); got != tc.want {
				t.Errorf("normalizeModuleVersion(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
