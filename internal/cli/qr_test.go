package cli

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOrderedSignals(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want [][2]string
	}{
		{
			name: "preserves authored order, not lexical",
			raw:  `{"Placement":"Bus stop","City":"Copenhagen"}`,
			want: [][2]string{{"Placement", "Bus stop"}, {"City", "Copenhagen"}},
		},
		{
			name: "empty map yields no pairs",
			raw:  `{}`,
			want: nil,
		},
		{
			name: "absent field yields no pairs",
			raw:  ``,
			want: nil,
		},
		{
			name: "null yields no pairs",
			raw:  `null`,
			want: nil,
		},
		{
			name: "malformed json is swallowed on this display path",
			raw:  `{"City":`,
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := orderedSignals(json.RawMessage(tc.raw))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("orderedSignals(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
