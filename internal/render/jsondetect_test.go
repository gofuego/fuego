package render

import "testing"

func TestJSONDetection(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html": `<body>{{block "content" .}}{{.Page.Content}}{{end}}</body>`,
		// References .JSON directly in an action.
		"layouts/hydrated.html": `{{define "content"}}<script id="fuego-data">{{.JSON}}</script>{{end}}`,
		// References $.JSON from inside a range.
		"layouts/dollar.html": `{{define "content"}}{{range .Page.Envelope.items}}{{$.JSON}}{{end}}{{end}}`,
		// References .JSON only inside a condition.
		"layouts/conditional.html": `{{define "content"}}{{if .JSON}}has data{{end}}{{end}}`,
		// No .JSON anywhere; envelope key named JSON-ish must not confuse it.
		"layouts/plain.html": `{{define "content"}}<p>{{.Page.Envelope.jsonish}}</p>{{end}}`,
	})

	tc, err := LoadTemplates(theme)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		layout string
		want   bool
	}{
		{"hydrated", true},
		{"dollar", true},
		{"conditional", true},
		{"plain", false},
		{"", false},          // bare base
		{"missing", false},   // falls back to base
	}
	for _, c := range cases {
		if got := tc.UsesJSON(tc.GetLayout(c.layout)); got != c.want {
			t.Errorf("UsesJSON(layout %q) = %v, want %v", c.layout, got, c.want)
		}
	}
}

func TestJSONDetectionInBase(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html":           `<script>{{.JSON}}</script>{{block "content" .}}{{end}}`,
		"layouts/any.html":    `{{define "content"}}x{{end}}`,
	})

	tc, err := LoadTemplates(theme)
	if err != nil {
		t.Fatal(err)
	}

	// Base references .JSON, so every layout cloned from it inherits the reference.
	if !tc.UsesJSON(tc.GetLayout("")) {
		t.Error("base with .JSON should be detected")
	}
	if !tc.UsesJSON(tc.GetLayout("any")) {
		t.Error("layout cloned from .JSON-using base should be detected")
	}
}
