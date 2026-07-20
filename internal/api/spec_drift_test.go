package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// clientEndpoints lists every (method, path) the hand-written client calls.
// The test asserts each one exists in the vendored OpenAPI snapshot, so a
// Core route rename/removal surfaces as a test failure after `zeb spec sync`
// instead of a runtime 404. Add a row when adding a client method.
var clientEndpoints = []struct {
	Method string
	Path   string
}{
	{"get", "/api/v1/health"},
	{"get", "/api/v1/me"},
	{"get", "/api/v1/spaces/{spaceId}/domains"},
	{"get", "/api/v1/spaces/{spaceId}/collections"},
	{"post", "/api/v1/spaces/{spaceId}/collections"},
	{"get", "/api/v1/spaces/{spaceId}/collections/{collectionId}"},
	{"patch", "/api/v1/spaces/{spaceId}/collections/{collectionId}"},
	{"delete", "/api/v1/spaces/{spaceId}/collections/{collectionId}"},
	{"post", "/api/v1/spaces/{spaceId}/collections/{collectionId}/convert-to-manual"},
	{"get", "/api/v1/spaces/{spaceId}/collections/{collectionId}/links"},
	{"post", "/api/v1/spaces/{spaceId}/collections/{collectionId}/links"},
	{"delete", "/api/v1/spaces/{spaceId}/collections/{collectionId}/links"},
	{"get", "/api/v1/spaces/{spaceId}/links"},
	{"post", "/api/v1/spaces/{spaceId}/links"},
	{"post", "/api/v1/spaces/{spaceId}/links/query"},
	{"get", "/api/v1/spaces/{spaceId}/links/lookup"},
	{"get", "/api/v1/spaces/{spaceId}/links/{linkId}"},
	{"patch", "/api/v1/spaces/{spaceId}/links/{linkId}"},
	{"delete", "/api/v1/spaces/{spaceId}/links/{linkId}"},
	{"post", "/api/v1/spaces/{spaceId}/links/bulk"},
	{"delete", "/api/v1/spaces/{spaceId}/links/bulk"},
	{"get", "/api/v1/spaces/{spaceId}/links/{linkId}/qr-variants"},
	{"get", "/api/v1/spaces/{spaceId}/links/{linkId}/qr/image"},
	{"post", "/api/v1/spaces/{spaceId}/links/{linkId}/qr/export"},
}

func loadSpecPaths(t *testing.T) map[string]map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "openapi", "openapi.json"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse snapshot: %v", err)
	}
	if len(spec.Paths) == 0 {
		t.Fatal("snapshot has no paths; run `zeb spec sync`")
	}
	return spec.Paths
}

func TestClientEndpointsExistInSpec(t *testing.T) {
	paths := loadSpecPaths(t)
	for _, endpoint := range clientEndpoints {
		operations, ok := paths[endpoint.Path]
		if !ok {
			t.Errorf("client uses %s %s but the snapshot has no such path", endpoint.Method, endpoint.Path)
			continue
		}
		if _, ok := operations[endpoint.Method]; !ok {
			t.Errorf("client uses %s %s but the snapshot path lacks that method", endpoint.Method, endpoint.Path)
		}
	}
}

// TestSpecOperationsNotInClient documents which spec operations the CLI does
// not consume yet. It fails when the API grows a NEW operation the CLI team
// has not looked at — extend the client (and clientEndpoints) or add the op
// to knownUnimplemented with a reason.
func TestSpecOperationsNotInClient(t *testing.T) {
	knownUnimplemented := map[string]string{
		"patch /api/v1/spaces/{spaceId}/links/bulk":      "bulk update: no CLI verb needs it yet (links update covers single)",
		"post /api/v1/spaces/{spaceId}/collections/bulk": "bulk collection create: niche for a CLI; `zeb collection create` covers the flow",
		// QR design authoring (styles, signals) is a studio flow, not a CLI one.
		// The CLI reads/exports/renders QR (qr-variants list, qr/image, qr/export);
		// it deliberately does not create or edit variant designs.
		"post /api/v1/spaces/{spaceId}/links/{linkId}/qr-variants":                  "QR variant authoring is a studio flow; CLI lists variants and exports/renders them",
		"get /api/v1/spaces/{spaceId}/links/{linkId}/qr-variants/{variantId}":       "single-variant detail: `zeb qr variants` lists them with ids and image URLs",
		"patch /api/v1/spaces/{spaceId}/links/{linkId}/qr-variants/{variantId}":     "QR variant authoring is a studio flow",
		"delete /api/v1/spaces/{spaceId}/links/{linkId}/qr-variants/{variantId}":    "QR variant authoring is a studio flow",
		"get /api/v1/spaces/{spaceId}/links/{linkId}/qr-variants/{variantId}/image": "variant image: `zeb qr <link> --download --variant <id>` renders any variant via qr/image",
	}
	implemented := map[string]bool{}
	for _, endpoint := range clientEndpoints {
		implemented[endpoint.Method+" "+endpoint.Path] = true
	}
	for path, operations := range loadSpecPaths(t) {
		for method := range operations {
			if method == "parameters" {
				continue
			}
			key := method + " " + path
			if implemented[key] {
				continue
			}
			if _, known := knownUnimplemented[key]; known {
				continue
			}
			t.Errorf("spec has %s with no client method and no knownUnimplemented entry — wire it or record the decision", key)
		}
	}
}
