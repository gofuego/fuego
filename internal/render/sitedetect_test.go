package render

import "testing"

func TestUsesSitePages(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html": `<body>{{block "content" .}}{{.Page.Content}}{{end}}</body>`,

		// Direct .Site.Pages use.
		"layouts/direct.html": `{{define "content"}}{{range .Site.Pages}}{{.URL}}{{end}}{{end}}`,
		// Via where/sortBy pipeline.
		"layouts/pipeline.html": `{{define "content"}}{{range sortBy (where .Site.Pages "type" "x") "url"}}{{.URL}}{{end}}{{end}}`,
		// Via a partial.
		"layouts/viapartial.html": `{{define "content"}}{{partial "nav" .}}{{end}}`,
		// .Site.Name only — a build constant, NOT content-dependent.
		"layouts/siteblind.html": `{{define "content"}}<h1>{{.Site.Name}}</h1>{{.Page.Content}}{{end}}`,
		// Plain.
		"layouts/plain.html": `{{define "content"}}{{.Page.Content}}{{end}}`,

		// Partials: nav uses .Site.Pages; outer calls nav (transitive); footer is blind.
		"partials/nav.html":    `<nav>{{range .Site.Pages}}{{.URL}}{{end}}</nav>`,
		"partials/outer.html":  `{{partial "nav" .}}`,
		"partials/footer.html": `<footer>{{.Site.Name}}</footer>`,
	})

	tc, err := LoadTemplates(theme, nil)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		layout string
		want   bool
	}{
		{"direct", true},
		{"pipeline", true},
		{"viapartial", true},
		{"siteblind", false},
		{"plain", false},
		{"", false}, // bare base
	}
	for _, c := range cases {
		if got := tc.UsesSitePages(tc.GetLayout(c.layout)); got != c.want {
			t.Errorf("UsesSitePages(layout %q) = %v, want %v", c.layout, got, c.want)
		}
	}
}

func TestUsesSitePagesTransitivePartial(t *testing.T) {
	theme := writeTheme(t, map[string]string{
		"base.html":            `<body>{{partial "outer" .}}{{block "content" .}}{{end}}</body>`,
		"layouts/any.html":     `{{define "content"}}x{{end}}`,
		"partials/outer.html":  `{{partial "inner" .}}`,
		"partials/inner.html":  `{{range .Site.Pages}}{{.URL}}{{end}}`,
	})

	tc, err := LoadTemplates(theme, nil)
	if err != nil {
		t.Fatal(err)
	}
	// base calls outer → inner → .Site.Pages, so every page is content-dependent.
	if !tc.UsesSitePages(tc.GetLayout("any")) {
		t.Error("transitive partial chain to .Site.Pages not detected")
	}
}
