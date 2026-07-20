package cli

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zeb-link/zeb/internal/api"
)

func buildAnalyticsForTest(t *testing.T, args []string) (api.AnalyticsQueryInput, error) {
	t.Helper()
	flags := &analyticsFlags{}
	cmd := &cobra.Command{Use: "analytics"}
	addAnalyticsFlags(cmd, flags)
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags(%v): %v", args, err)
	}
	return buildAnalyticsInput(cmd, flags, cmd.Flags().Args())
}

func TestBuildAnalyticsInput(t *testing.T) {
	t.Run("shared object scope maps like links query", func(t *testing.T) {
		in, err := buildAnalyticsForTest(t, []string{"--status", "active", "--not", "clicked", "--group-by", "browser"})
		if err != nil {
			t.Fatal(err)
		}
		if in.Status != "active" || in.GroupBy != "browser" {
			t.Errorf("status/groupBy = %q/%q", in.Status, in.GroupBy)
		}
		if !reflect.DeepEqual(in.Negate, []string{"clicked"}) {
			t.Errorf("negate = %v", in.Negate)
		}
	})

	t.Run("click dims + measure/range", func(t *testing.T) {
		in, err := buildAnalyticsForTest(t, []string{"--country", "US,GB", "--measure", "uniqueClicks", "--range", "30d"})
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(in.Country, []string{"US", "GB"}) {
			t.Errorf("country = %v", in.Country)
		}
		if in.Measure != "uniqueClicks" || in.Range != "30d" {
			t.Errorf("measure/range = %q/%q", in.Measure, in.Range)
		}
	})

	t.Run("from-qr tri-state", func(t *testing.T) {
		on, _ := buildAnalyticsForTest(t, []string{"--from-qr"})
		if on.FromQr == nil || *on.FromQr != true {
			t.Errorf("--from-qr should set true, got %v", on.FromQr)
		}
		off, _ := buildAnalyticsForTest(t, []string{"--from-qr=false"})
		if off.FromQr == nil || *off.FromQr != false {
			t.Errorf("--from-qr=false should set false, got %v", off.FromQr)
		}
		unset, _ := buildAnalyticsForTest(t, []string{"--group-by", "country"})
		if unset.FromQr != nil {
			t.Errorf("unset --from-qr should be nil, got %v", unset.FromQr)
		}
	})

	t.Run("collection scope", func(t *testing.T) {
		in, err := buildAnalyticsForTest(t, []string{"--collection", "col_123", "--group-by", "country"})
		if err != nil {
			t.Fatal(err)
		}
		if in.CollectionID != "col_123" {
			t.Errorf("collectionId = %q", in.CollectionID)
		}
	})

	t.Run("filter is exclusive with dimension flags", func(t *testing.T) {
		if _, err := buildAnalyticsForTest(t, []string{"--filter", `{"country":["US"]}`, "--status", "active"}); err == nil {
			t.Error("expected an error mixing --filter with a dimension flag")
		}
	})

	t.Run("in-collection and uncollected conflict", func(t *testing.T) {
		if _, err := buildAnalyticsForTest(t, []string{"--in-collection", "--uncollected"}); err == nil {
			t.Error("expected an error for --in-collection + --uncollected")
		}
	})

	t.Run("link with click dimensions is fine (this link's clicks, broken down)", func(t *testing.T) {
		if _, err := buildAnalyticsForTest(t, []string{"--link", "zbrah.link/2xc", "--group-by", "country", "--range", "7d"}); err != nil {
			t.Errorf("--link + click dims should be allowed, got %v", err)
		}
	})

	t.Run("link is exclusive with free text", func(t *testing.T) {
		if _, err := buildAnalyticsForTest(t, []string{"--link", "zbrah.link/2xc", "some search"}); err == nil {
			t.Error("expected an error mixing --link with free text")
		}
	})

	t.Run("link is exclusive with the object-scope filters", func(t *testing.T) {
		if _, err := buildAnalyticsForTest(t, []string{"--link", "zbrah.link/2xc", "--status", "active"}); err == nil {
			t.Error("expected an error mixing --link with an object-scope filter")
		}
	})

	t.Run("link is exclusive with collection", func(t *testing.T) {
		if _, err := buildAnalyticsForTest(t, []string{"--link", "zbrah.link/2xc", "--collection", "col_123"}); err == nil {
			t.Error("expected an error mixing --link with --collection")
		}
	})

	t.Run("a link-shaped positional is refused with a --link hint (not a silent zero)", func(t *testing.T) {
		linkish := []string{
			"http://kerns.local:3001/ywv",    // full URL with scheme + port
			"https://zbrah.link/2xc",         // full https URL
			"zbrah.link/2xc",                 // bare host/code short link
			"kerns.local:3001/ywv",           // host:port/code, no scheme
			"lnk_01kxjvtj3sphpfqbta3pcgecvs", // a link id
		}
		for _, arg := range linkish {
			if _, err := buildAnalyticsForTest(t, []string{arg}); err == nil {
				t.Errorf("%q should be refused as link-shaped, got no error", arg)
			}
		}
	})

	t.Run("a genuine search term still searches", func(t *testing.T) {
		ok := []string{
			"cnn",          // a plain word
			"cnn.com",      // a bare host, no path — could be a real search
			"promo launch", // a multi-word phrase
			"foo/bar",      // a slash but no dotted host — not a link
		}
		for _, arg := range ok {
			in, err := buildAnalyticsForTest(t, splitArgs(arg))
			if err != nil {
				t.Errorf("%q should search, got error %v", arg, err)
				continue
			}
			if in.Query != arg {
				t.Errorf("%q should map to Query, got %q", arg, in.Query)
			}
		}
	})
}

func splitArgs(s string) []string {
	return strings.Fields(s)
}
