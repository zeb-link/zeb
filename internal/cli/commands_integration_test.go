package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeb-link/zeb/internal/config"
)

// testHarness runs the real command tree against an httptest Core with an
// isolated $HOME, so command wiring (chunking, fallbacks, flags) is exercised
// end-to-end without a live server.
func testHarness(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("HOME", t.TempDir())
	for _, env := range []string{"ZLINK_API_KEY", "ZLINK_API_URL", "ZLINK_COLLECTION", "ZLINK_DOMAIN", "ZLINK_SPACE"} {
		t.Setenv(env, "")
	}
	if err := config.SaveCredentials("zeb_test"); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveConfig(config.Config{
		APIURL:      server.URL + "/api/v1",
		ActiveSpace: "spc_00000000000000000000000000",
	}); err != nil {
		t.Fatal(err)
	}
	return server
}

func runZeb(t *testing.T, args ...string) error {
	t.Helper()
	cmd := newRootCommand(&rootOptions{Version: "test"})
	cmd.SetArgs(args)
	cmd.SetOut(&strings.Builder{})
	cmd.SetErr(&strings.Builder{})
	return cmd.Execute()
}

func TestLinksDeleteChunksAt250(t *testing.T) {
	var batchSizes []int
	server := testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !strings.HasSuffix(r.URL.Path, "/links/bulk") {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		var body struct {
			LinkIDs []string `json:"linkIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		batchSizes = append(batchSizes, len(body.LinkIDs))
		results := make([]map[string]any, len(body.LinkIDs))
		for i, id := range body.LinkIDs {
			results[i] = map[string]any{"linkId": id, "success": true}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))
	_ = server

	ids := make([]string, 0, 301)
	for i := 0; i < 301; i++ {
		ids = append(ids, fmt.Sprintf("lnk_%026d", i))
	}
	if err := runZeb(t, append([]string{"links", "delete", "--json"}, ids...)...); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if len(batchSizes) != 2 || batchSizes[0] != 250 || batchSizes[1] != 51 {
		t.Fatalf("batch sizes = %v, want [250 51]", batchSizes)
	}
}

func TestLinksDeleteAllRowsFailedExitsNonZero(t *testing.T) {
	testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			LinkIDs []string `json:"linkIds"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		results := make([]map[string]any, len(body.LinkIDs))
		for i, id := range body.LinkIDs {
			results[i] = map[string]any{
				"linkId": id, "success": false,
				"error": map[string]string{"code": "READ_ONLY", "message": "workspace is read-only"},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
	}))

	err := runZeb(t, "links", "delete", "--json", "lnk_00000000000000000000000000")
	if err == nil || !strings.Contains(err.Error(), "READ_ONLY") {
		t.Fatalf("expected READ_ONLY failure, got %v", err)
	}
}

func TestLinksDeleteRejectsNonLinkIDs(t *testing.T) {
	testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("no request should be made for invalid ids")
	}))
	err := runZeb(t, "links", "delete", "col_notalink")
	if err == nil || !strings.Contains(err.Error(), "does not look like a link id") {
		t.Fatalf("expected id-shape error, got %v", err)
	}
}

func TestBulkCreateReportsPartialFailure(t *testing.T) {
	testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/links/bulk") && r.Method == http.MethodPost:
			var body struct {
				Items []struct {
					TargetURL string `json:"targetUrl"`
				} `json:"items"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			results := make([]map[string]any, len(body.Items))
			for i, item := range body.Items {
				if i == 1 {
					results[i] = map[string]any{
						"index": i, "success": false,
						"error": map[string]string{"code": "PATH_TAKEN", "message": "path exists"},
					}
					continue
				}
				results[i] = map[string]any{
					"index": i, "success": true,
					"link": map[string]any{
						"id": fmt.Sprintf("lnk_%026d", i), "spaceId": "spc_00000000000000000000000000",
						"hostname": "z.dk", "path": fmt.Sprintf("p%d", i),
						"shortUrl": fmt.Sprintf("https://z.dk/p%d", i), "targetUrl": item.TargetURL,
						"isActive": true, "createdAt": "2026-07-07T00:00:00Z",
					},
				}
			}
			w.WriteHeader(http.StatusMultiStatus)
			_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))

	// Partial failure: command exits zero, failures are in the report.
	if err := runZeb(t, "links", "create", "--json", "--no-collection", "https://a.com", "https://b.com", "https://c.com"); err != nil {
		t.Fatalf("partial failure should not fail the command: %v", err)
	}
}

func TestCreateFallsBackWhenAmbientCollectionIsGone(t *testing.T) {
	var createdCollectionRef *string
	testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/collections") && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"collections": []any{}})
		case strings.HasSuffix(r.URL.Path, "/links") && r.Method == http.MethodPost:
			var body struct {
				Collection *string `json:"collection"`
				TargetURL  string  `json:"targetUrl"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			createdCollectionRef = body.Collection
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"link": map[string]any{
					"id": "lnk_00000000000000000000000001", "spaceId": "spc_00000000000000000000000000",
					"hostname": "z.dk", "path": "x", "shortUrl": "https://z.dk/x",
					"targetUrl": body.TargetURL, "isActive": true, "createdAt": "2026-07-07T00:00:00Z",
				},
				"source": "api", "targetReachable": nil,
			})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))

	// Saved context references a collection that no longer exists (post-wipe).
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.ActiveCollection = "col_00000000000000000000000000"
	if err := config.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	if err := runZeb(t, "links", "create", "--json", "https://a.com"); err != nil {
		t.Fatalf("ambient stale collection must not fail the create: %v", err)
	}
	if createdCollectionRef != nil && *createdCollectionRef != "" {
		t.Fatalf("create should have dropped the dead collection, sent %q", *createdCollectionRef)
	}
}

func TestCreateFailsWhenExplicitCollectionIsGone(t *testing.T) {
	testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/collections") && r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode(map[string]any{"collections": []any{}})
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))

	err := runZeb(t, "links", "create", "--collection", "ghost", "https://a.com")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("explicit dead collection must fail the create, got %v", err)
	}
}

func TestLinksListAllFollowsPagination(t *testing.T) {
	var cursorsSeen []string
	testHarness(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/links") {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		cursor := r.URL.Query().Get("cursor")
		cursorsSeen = append(cursorsSeen, cursor)
		w.Header().Set("Content-Type", "application/json")
		link := map[string]any{
			"id": "lnk_00000000000000000000000001", "spaceId": "spc_00000000000000000000000000",
			"hostname": "z.dk", "path": "x", "shortUrl": "https://z.dk/x",
			"targetUrl": "https://a.com", "isActive": true, "createdAt": "2026-07-07T00:00:00Z",
		}
		if cursor == "" {
			_ = json.NewEncoder(w).Encode(map[string]any{"links": []any{link, link}, "nextCursor": "page2"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"links": []any{link}, "nextCursor": nil})
	}))

	if err := runZeb(t, "links", "--all", "--json"); err != nil {
		t.Fatal(err)
	}
	if len(cursorsSeen) != 2 || cursorsSeen[0] != "" || cursorsSeen[1] != "page2" {
		t.Fatalf("cursors = %v, want [\"\" page2]", cursorsSeen)
	}
}
