package render

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/FabioSol/fuego/core"
)

// pageRefLike mirrors the shape .Site.Pages refs will have: exported fields
// plus an Envelope map. where/sortBy must resolve keys against both.
type pageRefLike struct {
	URL      string
	Type     string
	Envelope core.Envelope
}

var refFixture = []pageRefLike{
	{URL: "/b", Type: "doc", Envelope: core.Envelope{"title": "Beta", "weight": 2}},
	{URL: "/a", Type: "doc", Envelope: core.Envelope{"title": "Alpha", "weight": 10}},
	{URL: "/c", Type: "post", Envelope: core.Envelope{"title": "Gamma", "weight": 1}},
}

func TestWhereStructField(t *testing.T) {
	out, err := whereFunc(refFixture, "type", "doc")
	if err != nil {
		t.Fatal(err)
	}
	got := out.([]pageRefLike)
	if len(got) != 2 || got[0].URL != "/b" || got[1].URL != "/a" {
		t.Errorf("where by struct field: got %+v", got)
	}
}

func TestWhereEnvelopeKey(t *testing.T) {
	out, err := whereFunc(refFixture, "title", "Gamma")
	if err != nil {
		t.Fatal(err)
	}
	if got := out.([]pageRefLike); len(got) != 1 || got[0].URL != "/c" {
		t.Errorf("where by envelope key: got %+v", got)
	}
}

func TestWhereLooseEquality(t *testing.T) {
	// Envelope numbers arrive as int from YAML; template literals are strings.
	out, err := whereFunc(refFixture, "weight", "2")
	if err != nil {
		t.Fatal(err)
	}
	if got := out.([]pageRefLike); len(got) != 1 || got[0].URL != "/b" {
		t.Errorf("loose equality int vs string: got %+v", got)
	}
}

func TestWhereMaps(t *testing.T) {
	maps := []map[string]any{
		{"kind": "x", "n": 1},
		{"kind": "y", "n": 2},
	}
	out, err := whereFunc(maps, "kind", "y")
	if err != nil {
		t.Fatal(err)
	}
	if got := out.([]map[string]any); len(got) != 1 || got[0]["n"] != 2 {
		t.Errorf("where over maps: got %+v", got)
	}
}

func TestWhereNotASlice(t *testing.T) {
	if _, err := whereFunc("nope", "k", "v"); err == nil {
		t.Error("expected error for non-slice collection")
	}
}

func TestSortByNumericAndOrder(t *testing.T) {
	out, err := sortByFunc(refFixture, "weight")
	if err != nil {
		t.Fatal(err)
	}
	got := out.([]pageRefLike)
	if got[0].URL != "/c" || got[1].URL != "/b" || got[2].URL != "/a" {
		t.Errorf("numeric asc: got %+v", got)
	}

	out, err = sortByFunc(refFixture, "weight", "desc")
	if err != nil {
		t.Fatal(err)
	}
	got = out.([]pageRefLike)
	if got[0].URL != "/a" || got[2].URL != "/c" {
		t.Errorf("numeric desc: got %+v", got)
	}

	if _, err := sortByFunc(refFixture, "weight", "sideways"); err == nil {
		t.Error("expected error for invalid order")
	}
}

func TestSortByStringAndStability(t *testing.T) {
	out, err := sortByFunc(refFixture, "title")
	if err != nil {
		t.Fatal(err)
	}
	got := out.([]pageRefLike)
	if got[0].Envelope["title"] != "Alpha" || got[2].Envelope["title"] != "Gamma" {
		t.Errorf("string sort: got %+v", got)
	}
}

func TestSortByDoesNotMutateInput(t *testing.T) {
	input := []pageRefLike{
		{URL: "/2", Envelope: core.Envelope{"weight": 2}},
		{URL: "/1", Envelope: core.Envelope{"weight": 1}},
	}
	if _, err := sortByFunc(input, "weight"); err != nil {
		t.Fatal(err)
	}
	if input[0].URL != "/2" {
		t.Error("sortBy mutated the input slice; shared slices like .Site.Pages must stay untouched")
	}
}

func TestLimitAndFirst(t *testing.T) {
	out, err := limitFunc(2, refFixture)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.([]pageRefLike); len(got) != 2 {
		t.Errorf("limit 2: got %d elements", len(got))
	}

	out, err = limitFunc(99, refFixture)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.([]pageRefLike); len(got) != 3 {
		t.Errorf("limit beyond length: got %d elements", len(got))
	}

	f, err := firstFunc(refFixture)
	if err != nil {
		t.Fatal(err)
	}
	if f.(pageRefLike).URL != "/b" {
		t.Errorf("first: got %+v", f)
	}

	f, err = firstFunc([]pageRefLike{})
	if err != nil {
		t.Fatal(err)
	}
	if f != nil {
		t.Errorf("first of empty: expected nil, got %+v", f)
	}
}

func TestDict(t *testing.T) {
	m, err := dictFunc("a", 1, "b", "two")
	if err != nil {
		t.Fatal(err)
	}
	if m["a"] != 1 || m["b"] != "two" {
		t.Errorf("dict: got %+v", m)
	}

	if _, err := dictFunc("a"); err == nil {
		t.Error("expected error for odd argument count")
	}
	if _, err := dictFunc(1, "a"); err == nil {
		t.Error("expected error for non-string key")
	}
}

func TestDateFormat(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{"2026-06-01", "Jun 1, 2026"},
		{"2026-06-01T15:04:05Z", "Jun 1, 2026"},
		{"2026-06-01 15:04:05", "Jun 1, 2026"},
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "Jun 1, 2026"},
	}
	for _, c := range cases {
		got, err := dateFormatFunc("Jan 2, 2006", c.in)
		if err != nil {
			t.Errorf("dateFormat(%v): %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("dateFormat(%v) = %q, want %q", c.in, got, c.want)
		}
	}

	if _, err := dateFormatFunc("Jan 2, 2006", "not-a-date"); err == nil {
		t.Error("expected error for unparseable date")
	}
	if _, err := dateFormatFunc("Jan 2, 2006", 42); err == nil {
		t.Error("expected error for unsupported type")
	}
}

// writeTheme builds a minimal theme dir for partial-loading tests.
func writeTheme(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestPartialExecution(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html":          `<body>{{partial "greet" .Page.Envelope.name}}</body>`,
		"partials/greet.html": `<p>Hello, {{.}}!</p>`,
	})

	tc, err := LoadTemplates(theme)
	if err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	data := TemplateData{Page: PageTemplateData{Envelope: core.Envelope{"name": "Fuego"}}}
	if err := tc.GetLayout("").Execute(&sb, data); err != nil {
		t.Fatalf("executing base with partial: %v", err)
	}
	if !strings.Contains(sb.String(), "<p>Hello, Fuego!</p>") {
		t.Errorf("partial output missing: %q", sb.String())
	}
}

func TestPartialCallsPartial(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html":           `{{partial "outer" .}}`,
		"partials/outer.html": `<nav>{{partial "inner" "x"}}</nav>`,
		"partials/inner.html": `<span>{{.}}</span>`,
	})

	tc, err := LoadTemplates(theme)
	if err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	if err := tc.GetLayout("").Execute(&sb, TemplateData{}); err != nil {
		t.Fatalf("nested partial: %v", err)
	}
	if sb.String() != `<nav><span>x</span></nav>` {
		t.Errorf("nested partial output: %q", sb.String())
	}
}

func TestPartialMissingNamesAvailable(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html":          `{{partial "nope" .}}`,
		"partials/nav.html":  `<nav/>`,
		"partials/foot.html": `<footer/>`,
	})

	tc, err := LoadTemplates(theme)
	if err != nil {
		t.Fatal(err)
	}

	err = tc.GetLayout("").Execute(&sb{}, TemplateData{})
	if err == nil {
		t.Fatal("expected error for missing partial")
	}
	for _, want := range []string{`"nope"`, "foot, nav"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should contain %q", err.Error(), want)
		}
	}
}

// sb is a throwaway writer for error-path executions.
type sb struct{}

func (*sb) Write(p []byte) (int, error) { return len(p), nil }

func TestPartialAvailableInLayoutAndRenderer(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html":             `{{block "content" .}}base{{end}}`,
		"layouts/doc.html":      `{{define "content"}}{{partial "tag" "layout"}}{{end}}`,
		"renderers/note.html":   `{{partial "tag" .Content}}`,
		"partials/tag.html":     `[{{.}}]`,
	})

	tc, err := LoadTemplates(theme)
	if err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	if err := tc.GetLayout("doc").Execute(&out, TemplateData{}); err != nil {
		t.Fatalf("layout partial: %v", err)
	}
	if out.String() != "[layout]" {
		t.Errorf("layout partial output: %q", out.String())
	}

	html := string(tc.renderWithOverrides([]core.Node{{Type: "note", Content: "n1"}}))
	if html != "[n1]" {
		t.Errorf("renderer partial output: %q", html)
	}
}

func TestLookupKeyCaseInsensitiveField(t *testing.T) {
	got, ok := lookupKey(reflect.ValueOf(refFixture[0]), "url")
	if !ok || got != "/b" {
		t.Errorf("case-insensitive field lookup: got %v, %v", got, ok)
	}
}
