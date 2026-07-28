package cli

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/zeb-link/zeb/internal/openapi"
)

// The --sort help text advertises the API's sort vocabulary. The server stays
// the validator, but the hint must not drift from the live spec.
func TestLinkSortValuesMatchSpec(t *testing.T) {
	data, unreachable, err := openapi.FetchLiveSpec()
	if unreachable {
		t.Skipf("live spec unreachable (offline?): %v", err)
	}
	if err != nil {
		t.Fatalf("fetch live spec: %v", err)
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
		t.Fatalf("parse live spec: %v", err)
	}
	operations, ok := spec.Paths["/api/v1/spaces/{spaceId}/links"]
	if !ok {
		t.Fatal("live spec missing the list-links path")
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
	t.Fatal("live spec list-links GET has no sort parameter")
}
