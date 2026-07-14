package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zeb-link/zeb/internal/config"
)

func TestExpandRootURLShorthand(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want []string
	}{
		{"bare url", []string{"https://x.com"}, []string{"links", "create", "https://x.com"}},
		{"multiple urls", []string{"https://a.com", "http://b.com"}, []string{"links", "create", "https://a.com", "http://b.com"}},
		{"value flag before url", []string{"--domain", "z.dk", "https://x.com"}, []string{"links", "create", "--domain", "z.dk", "https://x.com"}},
		{"shorthand value flag before url", []string{"-t", "hi", "https://x.com"}, []string{"links", "create", "-t", "hi", "https://x.com"}},
		{"equals form flag before url", []string{"--title=hi", "https://x.com"}, []string{"links", "create", "--title=hi", "https://x.com"}},
		{"bool flag before url", []string{"--json", "https://x.com"}, []string{"links", "create", "--json", "https://x.com"}},
		{"bool shorthand before url", []string{"-j", "https://x.com"}, []string{"links", "create", "-j", "https://x.com"}},
		{"double dash then url", []string{"--", "https://x.com"}, []string{"links", "create", "--", "https://x.com"}},
		{"subcommand stays", []string{"links"}, []string{"links"}},
		{"subcommand with url arg stays", []string{"links", "create", "https://x.com"}, []string{"links", "create", "https://x.com"}},
		{"non-url positional stays", []string{"status"}, []string{"status"}},
		{"value flag consuming what looks like a url", []string{"--title", "https://x.com"}, []string{"--title", "https://x.com"}},
		{"empty", []string{}, []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newRootCommand(&rootOptions{})
			got := expandRootURLShorthand(cmd, tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("expand(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

// Every value-taking create/root flag must be detected as consuming the next
// argument — this is what keeps `zeb --newflag value https://…` working when
// a flag is added. Bool flags must NOT consume.
func TestFlagConsumesValueDerivedFromFlagSet(t *testing.T) {
	cmd := newRootCommand(&rootOptions{})
	consuming := []string{"--api-key", "--api-url", "--space", "-s", "--collection", "-c", "--domain", "-d", "--path", "--short-code", "--namespace", "--title", "-t"}
	for _, flag := range consuming {
		if !flagConsumesValue(cmd, flag) {
			t.Errorf("%s should consume the next argument", flag)
		}
	}
	notConsuming := []string{"--json", "-j", "--no-collection", "--no-verify", "--title=x", "-t=x", "--unknown", "-z"}
	for _, flag := range notConsuming {
		if flagConsumesValue(cmd, flag) {
			t.Errorf("%s should NOT consume the next argument", flag)
		}
	}
}

// --api-url is the owner-only escape hatch: it must keep working but never
// appear in help output.
func TestAPIURLFlagHiddenButFunctional(t *testing.T) {
	cmd := newRootCommand(&rootOptions{})
	flag := cmd.PersistentFlags().Lookup("api-url")
	if flag == nil {
		t.Fatal("api-url flag missing")
	}
	if !flag.Hidden {
		t.Fatal("api-url flag must be hidden")
	}
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "api-url") {
		t.Fatal("api-url leaked into --help output")
	}
}

// Login picks the environment fresh: hidden flag > env > built-in production
// default — the STORED config URL must play no part.
func TestLoginAPIURLResolution(t *testing.T) {
	t.Setenv("ZLINK_API_URL", "")

	url, overridden := loginAPIURL(&rootOptions{})
	if overridden || url != config.DefaultAPIURL() {
		t.Fatalf("no override: url=%q overridden=%v", url, overridden)
	}

	t.Setenv("ZLINK_API_URL", "http://localhost:3000")
	url, overridden = loginAPIURL(&rootOptions{})
	if !overridden || url != "http://localhost:3000/api/v1" {
		t.Fatalf("env override: url=%q overridden=%v", url, overridden)
	}

	url, overridden = loginAPIURL(&rootOptions{APIURL: "http://other:1234"})
	if !overridden || url != "http://other:1234/api/v1" {
		t.Fatalf("flag override beats env: url=%q overridden=%v", url, overridden)
	}
}

func TestTruncateIsRuneSafe(t *testing.T) {
	cases := []struct {
		value string
		limit int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly-10", 10, "exactly-10"},
		{"this is far too long", 10, "this is f…"},
		{"æøå-æøå-æøå", 8, "æøå-æøå…"},
		{"日本語のタイトルです", 5, "日本語の…"},
		{"anything", 1, "anything"},
	}
	for _, tc := range cases {
		if got := truncate(tc.value, tc.limit); got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.value, tc.limit, got, tc.want)
		}
	}
}
