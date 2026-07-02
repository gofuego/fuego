package buildcache

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gofuego/fuego/core"
)

func TestClonePageIsolatesMutation(t *testing.T) {
	orig := ParsedPage{
		ContentHash: "h",
		Envelope: core.Envelope{
			"title": "A",
			"tags":  []any{"go"},
			"meta":  map[string]any{"nested": []string{"x"}},
			"cards": []map[string]any{{"name": "n1"}},
		},
		Nodes: []core.Node{{
			Type:       "body",
			Content:    "original",
			Attributes: map[string]any{"k": "v"},
			Children:   []core.Node{{Type: "child", Content: "c"}},
		}},
	}
	snap := ClonePage(orig)

	// Mutate the original the way hooks mutate live pages.
	orig.Envelope["added"] = "hook-value"
	orig.Envelope["meta"].(map[string]any)["nested"] = append(
		orig.Envelope["meta"].(map[string]any)["nested"].([]string), "y")
	orig.Envelope["cards"].([]map[string]any)[0]["name"] = "mutated"
	orig.Nodes[0].Content = "rewritten"
	orig.Nodes[0].Attributes["k"] = "mutated"
	orig.Nodes[0].Children[0].Content = "mutated"

	if _, ok := snap.Envelope["added"]; ok {
		t.Error("hook-added envelope key leaked into the snapshot")
	}
	if got := snap.Envelope["meta"].(map[string]any)["nested"].([]string); len(got) != 1 || got[0] != "x" {
		t.Errorf("nested slice leaked: %v", got)
	}
	if got := snap.Envelope["cards"].([]map[string]any)[0]["name"]; got != "n1" {
		t.Errorf("nested map-in-slice leaked: %v", got)
	}
	if snap.Nodes[0].Content != "original" {
		t.Error("node content mutation leaked into the snapshot")
	}
	if snap.Nodes[0].Attributes["k"] != "v" {
		t.Error("node attribute mutation leaked into the snapshot")
	}
	if snap.Nodes[0].Children[0].Content != "c" {
		t.Error("child node mutation leaked into the snapshot")
	}
}

func TestSaveRoundTripsJSONShapedComposites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.bin")
	env := core.Envelope{
		"servers":  []map[string]any{{"name": "gh", "args": []string{"-y"}}},
		"numbers":  []int{3, 7},
		"floats":   []float64{1.5},
		"flags":    []bool{true},
		"labels":   map[string]string{"a": "b"},
		"rows":     []map[string]string{{"k": "v"}},
		"strs":     []string{"x"},
	}
	c := New(Header{BinaryID: "b", ConfigHash: "c", ThemeHash: "t"})
	c.Pages["mcp.json"] = ParsedPage{ContentHash: "h", Envelope: env}

	dropped, err := Save(path, c)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if len(dropped) != 0 {
		t.Fatalf("JSON-shaped composites should be cacheable, dropped %v", dropped)
	}
	got, ok := Load(path)
	if !ok {
		t.Fatal("expected cache to load")
	}
	if !reflect.DeepEqual(got.Pages["mcp.json"].Envelope, env) {
		t.Errorf("round-trip mismatch:\n got %#v\nwant %#v", got.Pages["mcp.json"].Envelope, env)
	}
}

// exotic is a package-private type gob cannot know about — the stand-in for a
// pack putting its own struct in an envelope.
type exotic struct{ X string }

func TestSaveDropsUnencodablePagesOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.bin")
	c := New(Header{BinaryID: "b", ConfigHash: "c", ThemeHash: "t"})
	c.Pages["good.md"] = ParsedPage{ContentHash: "h1", Envelope: core.Envelope{"title": "ok"}}
	c.Pages["bad.md"] = ParsedPage{ContentHash: "h2", Envelope: core.Envelope{"kv": []exotic{{X: "v"}}}}

	dropped, err := Save(path, c)
	if err != nil {
		t.Fatalf("Save should degrade per page, got error: %v", err)
	}
	if len(dropped) != 1 || dropped[0] != "bad.md" {
		t.Fatalf("expected exactly bad.md dropped, got %v", dropped)
	}

	got, ok := Load(path)
	if !ok {
		t.Fatal("expected cache to load")
	}
	if _, ok := got.Pages["good.md"]; !ok {
		t.Error("encodable page should survive an unencodable sibling")
	}
	if _, ok := got.Pages["bad.md"]; ok {
		t.Error("unencodable page should not be persisted")
	}
}
