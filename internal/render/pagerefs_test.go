package render

import (
	"testing"

	"github.com/gofuego/fuego/core"
)

func TestBuildPageRefsSortedAndSkipFiltered(t *testing.T) {
	pages := []*core.Page{
		{URL: "/c/", Type: "md", Envelope: core.Envelope{"title": "C"}},
		{URL: "/a/", Type: "md", Envelope: core.Envelope{"title": "A"}},
		{URL: "/b/", Type: "md", Skip: true, Envelope: core.Envelope{"title": "B"}},
	}

	refs := BuildPageRefs(pages)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs (skip excluded), got %d", len(refs))
	}
	if refs[0].URL != "/a/" || refs[1].URL != "/c/" {
		t.Errorf("refs not URL-sorted: %+v", refs)
	}
	if refs[0].Envelope["title"] != "A" {
		t.Errorf("envelope not carried: %+v", refs[0])
	}
}
