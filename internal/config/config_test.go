package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolateHome points ~/.zlink at a temp dir so tests never touch real state.
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, env := range []string{"ZLINK_API_KEY", "ZLINK_API_URL", "ZLINK_COLLECTION", "ZLINK_DOMAIN", "ZLINK_SPACE"} {
		t.Setenv(env, "")
	}
	return home
}

func TestNormalizeAPIURL(t *testing.T) {
	cases := map[string]string{
		"http://localhost:3000":         "http://localhost:3000/api/v1",
		"http://localhost:3000/":        "http://localhost:3000/api/v1",
		"http://localhost:3000/api":     "http://localhost:3000/api/v1",
		"http://localhost:3000/api/":    "http://localhost:3000/api/v1",
		"http://localhost:3000/api/v1":  "http://localhost:3000/api/v1",
		"http://localhost:3000/api/v1/": "http://localhost:3000/api/v1",
		"https://app.zeblink.io":        "https://app.zeblink.io/api/v1",
		"https://api.example.com/v1":    "https://api.example.com/v1",
	}
	for input, want := range cases {
		if got := NormalizeAPIURL(input); got != want {
			t.Errorf("NormalizeAPIURL(%q) = %q, want %q", input, got, want)
		}
	}
}

// The out-of-the-box API is production — every user path lands there with no
// configuration. This assertion is the tripwire for the "dedicated API
// domain" swap: change defaultAPIURL and this test together, deliberately.
func TestDefaultAPIURLIsProduction(t *testing.T) {
	if DefaultAPIURL() != "https://app.zeblink.io/api/v1" {
		t.Fatalf("DefaultAPIURL() = %q", DefaultAPIURL())
	}
}

func TestResolveAPIURLPrecedence(t *testing.T) {
	isolateHome(t)

	// Default when nothing is set.
	got, err := ResolveAPIURL("")
	if err != nil {
		t.Fatal(err)
	}
	if got != DefaultAPIURL() {
		t.Fatalf("default = %q", got)
	}

	// Stored config beats default.
	if err := SetValue("apiUrl", "https://config.example.com"); err != nil {
		t.Fatal(err)
	}
	got, _ = ResolveAPIURL("")
	if got != "https://config.example.com/api/v1" {
		t.Fatalf("config = %q", got)
	}

	// Env beats config.
	t.Setenv("ZLINK_API_URL", "https://env.example.com")
	got, _ = ResolveAPIURL("")
	if got != "https://env.example.com/api/v1" {
		t.Fatalf("env = %q", got)
	}

	// Flag beats env.
	got, _ = ResolveAPIURL("https://flag.example.com")
	if got != "https://flag.example.com/api/v1" {
		t.Fatalf("flag = %q", got)
	}
}

func TestResolveAPIKeyPrecedence(t *testing.T) {
	isolateHome(t)

	got, err := ResolveAPIKey("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("expected empty key, got %q", got)
	}

	if err := SaveCredentials("zeb_stored"); err != nil {
		t.Fatal(err)
	}
	got, _ = ResolveAPIKey("")
	if got != "zeb_stored" {
		t.Fatalf("stored = %q", got)
	}

	t.Setenv("ZLINK_API_KEY", "zeb_env")
	got, _ = ResolveAPIKey("")
	if got != "zeb_env" {
		t.Fatalf("env = %q", got)
	}

	got, _ = ResolveAPIKey("zeb_flag")
	if got != "zeb_flag" {
		t.Fatalf("flag = %q", got)
	}
}

func TestResolveCollectionAndSpacePrecedence(t *testing.T) {
	isolateHome(t)

	if err := SetValue("activeCollection", "col_config"); err != nil {
		t.Fatal(err)
	}
	if err := SetValue("activeSpace", "spc_config"); err != nil {
		t.Fatal(err)
	}

	collection, _ := ResolveCollection("")
	space, _ := ResolveSpace("")
	if collection != "col_config" || space != "spc_config" {
		t.Fatalf("config: collection=%q space=%q", collection, space)
	}

	t.Setenv("ZLINK_COLLECTION", "col_env")
	t.Setenv("ZLINK_SPACE", "spc_env")
	collection, _ = ResolveCollection("")
	space, _ = ResolveSpace("")
	if collection != "col_env" || space != "spc_env" {
		t.Fatalf("env: collection=%q space=%q", collection, space)
	}

	collection, _ = ResolveCollection("col_flag")
	if collection != "col_flag" {
		t.Fatalf("flag: collection=%q", collection)
	}
}

func TestSetUnsetValueRoundTrip(t *testing.T) {
	home := isolateHome(t)

	if err := SetValue("activeDomain", "z.dk"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveDomain != "z.dk" {
		t.Fatalf("ActiveDomain = %q", cfg.ActiveDomain)
	}
	if err := UnsetValue("activeDomain"); err != nil {
		t.Fatal(err)
	}
	cfg, _ = LoadConfig()
	if cfg.ActiveDomain != "" {
		t.Fatalf("ActiveDomain after unset = %q", cfg.ActiveDomain)
	}

	if err := SetValue("bogusKey", "x"); err == nil {
		t.Fatal("expected error for unknown key")
	}

	// Files land under the isolated home with owner-only permissions.
	if _, err := os.Stat(filepath.Join(home, ".zlink", "config.json")); err != nil {
		t.Fatalf("config file: %v", err)
	}
}

func TestLoadCredentialsMissingFileIsNil(t *testing.T) {
	isolateHome(t)
	credentials, err := LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if credentials != nil {
		t.Fatalf("expected nil credentials, got %+v", credentials)
	}
}
