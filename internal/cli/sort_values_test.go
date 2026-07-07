package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// The --sort help text advertises the API's sort vocabulary. The server stays
// the validator, but the hint must not drift from the snapshot.
func TestLinkSortValuesMatchSpec(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "openapi", "openapi.json"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name   string `json:"name"`
				Schema struct {
					Enum []string `json:"enum"`
				} `json:"schema"`
			} `json:"parameters"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse snapshot: %v", err)
	}
	operations, ok := spec.Paths["/api/v1/spaces/{spaceId}/links"]
	if !ok {
		t.Fatal("snapshot missing the list-links path")
	}
	for _, parameter := range operations["get"].Parameters {
		if parameter.Name != "sort" {
			continue
		}
		if !reflect.DeepEqual(parameter.Schema.Enum, linkSortValues) {
			t.Fatalf("linkSortValues = %v, spec sort enum = %v — update linkSortValues", linkSortValues, parameter.Schema.Enum)
		}
		return
	}
	t.Fatal("snapshot list-links GET has no sort parameter")
}
