package config

import (
	"reflect"
	"testing"
)

func TestMergeRules(t *testing.T) {
	tests := []struct {
		name   string
		layers []Layer
		want   map[string]any
	}{
		{
			name: "maps merge key-wise",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{"site": map[string]any{"name": "Pack", "base_url": "x"}}},
				{Source: "user", Data: map[string]any{"site": map[string]any{"name": "User"}}},
			},
			want: map[string]any{"site": map[string]any{"name": "User", "base_url": "x"}},
		},
		{
			name: "scalars replace",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{"port": 1}},
				{Source: "user", Data: map[string]any{"port": 2}},
			},
			want: map[string]any{"port": 2},
		},
		{
			name: "lists replace whole, never append",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{"ignore": []any{"a", "b"}}},
				{Source: "user", Data: map[string]any{"ignore": []any{"c"}}},
			},
			want: map[string]any{"ignore": []any{"c"}},
		},
		{
			name: "later pack beats earlier, user beats both",
			layers: []Layer{
				{Source: "p1", Data: map[string]any{"k": "one"}},
				{Source: "p2", Data: map[string]any{"k": "two"}},
				{Source: "user", Data: map[string]any{"k": "three"}},
			},
			want: map[string]any{"k": "three"},
		},
		{
			name: "list-in-map: a map value containing a list replaces whole",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{
					"parsers": map[string]any{"trivia": map[string]any{"rules": []any{"r1", "r2"}}},
				}},
				{Source: "user", Data: map[string]any{
					"parsers": map[string]any{"trivia": map[string]any{"rules": []any{"r3"}}},
				}},
			},
			want: map[string]any{
				"parsers": map[string]any{"trivia": map[string]any{"rules": []any{"r3"}}},
			},
		},
		{
			name: "map-in-list: lists of maps replace whole (no element merge)",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{"rules": []any{
					map[string]any{"match": "a"}, map[string]any{"match": "b"},
				}}},
				{Source: "user", Data: map[string]any{"rules": []any{
					map[string]any{"match": "z"},
				}}},
			},
			want: map[string]any{"rules": []any{map[string]any{"match": "z"}}},
		},
		{
			name: "scalar replaced by map when later layer deepens",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{"x": "scalar"}},
				{Source: "user", Data: map[string]any{"x": map[string]any{"deep": 1}}},
			},
			want: map[string]any{"x": map[string]any{"deep": 1}},
		},
		{
			name: "disjoint keys union",
			layers: []Layer{
				{Source: "pack", Data: map[string]any{"a": 1}},
				{Source: "user", Data: map[string]any{"b": 2}},
			},
			want: map[string]any{"a": 1, "b": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := mergeLayers(tt.layers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("merge = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestMergeDoesNotMutateLayers(t *testing.T) {
	packData := map[string]any{"site": map[string]any{"name": "Pack", "x": 1}}
	layers := []Layer{
		{Source: "pack", Data: packData},
		{Source: "user", Data: map[string]any{"site": map[string]any{"name": "User"}}},
	}
	mergeLayers(layers)

	if packData["site"].(map[string]any)["name"] != "Pack" {
		t.Error("merge mutated the pack layer's data")
	}
}

func TestProvenance(t *testing.T) {
	layers := []Layer{
		{Source: "p1", Data: map[string]any{
			"routes": map[string]any{"adr": "/old"},
			"only":   "p1",
		}},
		{Source: "user", Data: map[string]any{
			"routes": map[string]any{"adr": "/new"},
		}},
	}
	_, prov := mergeLayers(layers)

	if got := prov.Source("routes.adr"); got != "user" {
		t.Errorf("routes.adr provenance = %q, want user", got)
	}
	if got := prov.Source("only"); got != "p1" {
		t.Errorf("only provenance = %q, want p1", got)
	}
	if got := prov.Source("nonexistent"); got != "" {
		t.Errorf("missing key provenance = %q, want empty", got)
	}
}
