package api

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/zeb-link/zeb/internal/openapi"
)

// Response-shape drift, the class the endpoint tests cannot see: they check
// that a path still EXISTS, not that the body still has the fields we decode.
//
// Two rules, and the split matters:
//
//   - Every field our struct decodes must exist in the live spec. Always on,
//     never allowlisted — a Go tag with no matching property is a rename or a
//     removal, and it fails silently at runtime (the field decodes to its zero
//     value, so the CLI prints a blank instead of erroring).
//
//   - `mirrored` types must ALSO carry every property the spec defines. Opt-in,
//     because most responses are deliberately partial: Link ignores fallback,
//     the schedule fields, createdBy/createdVia and updatedAt, and that is a
//     product decision, not drift. Making the full rule global would need an
//     eight-entry allowlist on its first run, which just relocates the problem
//     into a list nobody reads. Mark a type mirrored when the CLI renders the
//     whole row and a new field is therefore something we want to be told about
//     — that is what caught `label` arriving on analytics rows.
type responseContract struct {
	Method  string
	Path    string
	Decodes reflect.Type
	// Mirrored: the CLI intends to carry every property this schema defines,
	// so a field added upstream should fail here rather than be dropped.
	Mirrored bool
}

var responseContracts = []responseContract{
	{
		Method:   "post",
		Path:     "/api/v1/spaces/{spaceId}/analytics/query",
		Decodes:  reflect.TypeOf(AnalyticsQueryResponse{}),
		Mirrored: true,
	},
	{
		Method:  "get",
		Path:    "/api/v1/spaces/{spaceId}/links",
		Decodes: reflect.TypeOf(ListLinksResponse{}),
	},
}

func loadSpecDocument(t *testing.T) map[string]any {
	t.Helper()
	data, unreachable, err := openapi.FetchLiveSpec()
	if unreachable {
		t.Skipf("live spec unreachable (offline?): %v", err)
	}
	if err != nil {
		t.Fatalf("fetch live spec: %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse live spec: %v", err)
	}
	return spec
}

// deref follows a local $ref into components; other schemas pass through.
func deref(spec map[string]any, schema map[string]any) map[string]any {
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return schema
	}
	components, ok := spec["components"].(map[string]any)
	if !ok {
		return schema
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		return schema
	}
	target, ok := schemas[strings.TrimPrefix(ref, prefix)].(map[string]any)
	if !ok {
		return schema
	}
	return target
}

// responseSchema resolves the 200 application/json schema for an operation.
func responseSchema(t *testing.T, spec map[string]any, method, path string) map[string]any {
	t.Helper()
	nest := func(m map[string]any, key string) map[string]any {
		next, _ := m[key].(map[string]any)
		return next
	}
	paths := nest(spec, "paths")
	operation := nest(nest(paths, path), method)
	if operation == nil {
		t.Fatalf("live spec has no %s %s — the endpoint test should have caught this", method, path)
	}
	schema := nest(nest(nest(nest(operation, "responses"), "200"), "content"), "application/json")
	schema = nest(schema, "schema")
	if schema == nil {
		t.Fatalf("%s %s has no 200 application/json schema", method, path)
	}
	return schema
}

// jsonName is the wire name of a struct field, or "" when it is not decoded.
func jsonName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "-" || tag == "" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	return name
}

// compare walks a Go type against its schema, reporting drift in both
// directions. `where` is a dotted path used only for readable failures.
func compare(t *testing.T, spec map[string]any, goType reflect.Type, schema map[string]any, mirrored bool, where string) {
	t.Helper()
	for goType.Kind() == reflect.Ptr {
		goType = goType.Elem()
	}
	schema = deref(spec, schema)

	if goType.Kind() == reflect.Slice {
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return // Not an array in the spec; the property check already ran.
		}
		compare(t, spec, goType.Elem(), items, mirrored, where+"[]")
		return
	}
	if goType.Kind() != reflect.Struct {
		return // Leaf: the property's presence is all we assert.
	}

	properties, _ := schema["properties"].(map[string]any)
	if properties == nil {
		return // Free-form object (details bags); nothing to compare.
	}

	decoded := map[string]bool{}
	for i := 0; i < goType.NumField(); i++ {
		field := goType.Field(i)
		name := jsonName(field)
		if name == "" {
			continue
		}
		decoded[name] = true
		property, ok := properties[name].(map[string]any)
		if !ok {
			t.Errorf("%s.%s is decoded by %s but the live spec no longer defines it — renamed or removed upstream, and it will decode to a zero value in silence",
				where, name, goType.Name())
			continue
		}
		compare(t, spec, field.Type, property, mirrored, where+"."+name)
	}

	if !mirrored {
		return
	}
	for name := range properties {
		if decoded[name] {
			continue
		}
		t.Errorf("%s.%s exists in the live spec but %s does not decode it — %s mirrors this schema, so either add the field or drop it from responseContracts",
			where, name, goType.Name(), goType.Name())
	}
}

func TestResponseStructsMatchSpec(t *testing.T) {
	spec := loadSpecDocument(t)
	for _, contract := range responseContracts {
		t.Run(contract.Method+" "+contract.Path, func(t *testing.T) {
			schema := responseSchema(t, spec, contract.Method, contract.Path)
			compare(t, spec, contract.Decodes, schema, contract.Mirrored, contract.Decodes.Name())
		})
	}
}
