package cli

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
)

// buildInputForTest parses args through a query command's flag set and runs
// buildQueryInput, exactly as the command does at runtime.
func buildInputForTest(t *testing.T, args []string) (api.QueryLinksInput, error) {
	t.Helper()
	flags := &queryLinksFlags{}
	cmd := &cobra.Command{Use: "query"}
	addQueryLinksFlags(cmd, flags)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags(%v): %v", args, err)
	}
	return buildQueryInput(cmd, flags, cmd.Flags().Args())
}

func boolPtr(b bool) *bool { return &b }

func TestBuildQueryInput(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want api.LinkFilter
	}{
		{
			name: "status + min-clicks maps to greaterThan",
			args: []string{"--status", "active", "--min-clicks", "5"},
			want: api.LinkFilter{Status: "active", Clicks: &api.ClickThreshold{Op: "greaterThan", Value: 5}},
		},
		{
			name: "max-clicks maps to lessThan",
			args: []string{"--max-clicks", "10"},
			want: api.LinkFilter{Clicks: &api.ClickThreshold{Op: "lessThan", Value: 10}},
		},
		{
			name: "free text becomes query",
			args: []string{"newsletter"},
			want: api.LinkFilter{Query: "newsletter"},
		},
		{
			name: "host lists OR within a dimension",
			args: []string{"--target-host", "cnn.com,bbc.com"},
			want: api.LinkFilter{TargetHost: []string{"cnn.com", "bbc.com"}},
		},
		{
			name: "not is passed through as negate",
			args: []string{"--status", "active", "--not", "clicked", "--not", "status"},
			want: api.LinkFilter{Status: "active", Negate: []string{"clicked", "status"}},
		},
		{
			name: "in-collection sets hasCollection true",
			args: []string{"--in-collection"},
			want: api.LinkFilter{HasCollection: boolPtr(true)},
		},
		{
			name: "uncollected sets hasCollection false",
			args: []string{"--uncollected"},
			want: api.LinkFilter{HasCollection: boolPtr(false)},
		},
		{
			name: "raw filter alone is parsed",
			args: []string{"--filter", `{"status":"inactive","targetHost":["cnn.com"]}`},
			want: api.LinkFilter{Status: "inactive", TargetHost: []string{"cnn.com"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input, err := buildInputForTest(t, tc.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(input.LinkFilter, tc.want) {
				t.Errorf("filter = %+v, want %+v", input.LinkFilter, tc.want)
			}
		})
	}
}

func TestBuildQueryInputErrors(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"both click thresholds", []string{"--min-clicks", "5", "--max-clicks", "10"}},
		{"both unique thresholds", []string{"--min-unique", "1", "--max-unique", "2"}},
		{"in-collection and uncollected", []string{"--in-collection", "--uncollected"}},
		{"filter mixed with a dimension flag", []string{"--filter", `{"status":"active"}`, "--status", "inactive"}},
		{"filter mixed with free text", []string{"--filter", `{"status":"active"}`, "text"}},
		{"invalid filter json", []string{"--filter", `{not json`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := buildInputForTest(t, tc.args); err == nil {
				t.Errorf("expected an error for args %v, got nil", tc.args)
			}
		})
	}
}

func TestIsEmptyFilter(t *testing.T) {
	if !isEmptyFilter(api.LinkFilter{}) {
		t.Error("zero filter should be empty")
	}
	if isEmptyFilter(api.LinkFilter{Status: "active"}) {
		t.Error("filter with a status should not be empty")
	}
	if isEmptyFilter(api.LinkFilter{HasCollection: boolPtr(false)}) {
		t.Error("filter with hasCollection=false should not be empty")
	}
}
