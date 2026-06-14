package index

import (
	"fmt"
	"testing"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

func listingPage(entries int) *core.Page {
	nodes := make([]core.Node, entries)
	for i := range nodes {
		nodes[i] = core.Node{Type: "page-ref", Attributes: map[string]any{"n": i}}
	}
	return &core.Page{
		RelPath:  "_virtual/collection/posts",
		URL:      "/posts/",
		Layout:   "listing",
		Type:     "collection",
		Envelope: core.Envelope{"title": "Posts"},
		Nodes:    nodes,
	}
}

func TestPaginateDisabledOrSmall(t *testing.T) {
	for _, size := range []int{0, 5, 10} {
		page := listingPage(5)
		out := paginate(page, size)
		if len(out) != 1 || out[0] != page {
			t.Errorf("page_size %d with 5 entries: expected single original page", size)
		}
		if size == 0 && out[0].Paginator != nil {
			t.Error("unpaginated page must have nil Paginator")
		}
	}
}

func TestPaginateSplits(t *testing.T) {
	page := listingPage(5)
	out := paginate(page, 2)

	if len(out) != 3 {
		t.Fatalf("5 entries / size 2: expected 3 pages, got %d", len(out))
	}

	// Page 1 keeps the base URL and the original pointer
	if out[0] != page || out[0].URL != "/posts/" {
		t.Errorf("page 1 should be the original page at /posts/, got %s", out[0].URL)
	}
	if out[1].URL != "/posts/page/2/" || out[2].URL != "/posts/page/3/" {
		t.Errorf("page URLs: %s, %s", out[1].URL, out[2].URL)
	}

	// Node distribution 2/2/1
	if len(out[0].Nodes) != 2 || len(out[1].Nodes) != 2 || len(out[2].Nodes) != 1 {
		t.Errorf("node split: %d/%d/%d", len(out[0].Nodes), len(out[1].Nodes), len(out[2].Nodes))
	}

	// Paginator chain
	for i, expect := range []core.Paginator{
		{CurrentPage: 1, TotalPages: 3, PrevURL: "", NextURL: "/posts/page/2/"},
		{CurrentPage: 2, TotalPages: 3, PrevURL: "/posts/", NextURL: "/posts/page/3/"},
		{CurrentPage: 3, TotalPages: 3, PrevURL: "/posts/page/2/", NextURL: ""},
	} {
		if *out[i].Paginator != expect {
			t.Errorf("page %d paginator = %+v, want %+v", i+1, *out[i].Paginator, expect)
		}
	}

	// Clones must not share envelopes; RelPaths must stay unique
	out[1].Envelope["title"] = "mutated"
	if out[0].Envelope["title"] != "Posts" {
		t.Error("page 2 envelope aliases page 1")
	}
	seen := map[string]bool{}
	for _, p := range out {
		if seen[p.RelPath] {
			t.Errorf("duplicate RelPath %s", p.RelPath)
		}
		seen[p.RelPath] = true
	}
}

func TestPaginateOrderPreserved(t *testing.T) {
	page := listingPage(7)
	out := paginate(page, 3)
	want := 0
	for _, p := range out {
		for _, n := range p.Nodes {
			if n.Attributes["n"] != want {
				t.Fatalf("entry order broken: got %v, want %d", n.Attributes["n"], want)
			}
			want++
		}
	}
	if want != 7 {
		t.Errorf("entries lost: saw %d of 7", want)
	}
}

func TestBuildCollectionsPaginated(t *testing.T) {
	var pages []*core.Page
	for i := 0; i < 5; i++ {
		pages = append(pages, &core.Page{
			RelPath:  fmt.Sprintf("posts/p%d.md", i),
			URL:      fmt.Sprintf("/posts/p%d/", i),
			Envelope: core.Envelope{"title": fmt.Sprintf("P%d", i), "order": i},
		})
	}

	virtual := BuildCollections(pages, map[string]config.CollectionConfig{"posts": {
		Match: "posts/**", SortBy: "order", Layout: "listing", Path: "/all-posts", PageSize: 2,
	}})

	if len(virtual) != 3 {
		t.Fatalf("expected 3 collection pages, got %d", len(virtual))
	}
	if virtual[0].URL != "/all-posts/" || virtual[1].URL != "/all-posts/page/2/" {
		t.Errorf("URLs: %s, %s", virtual[0].URL, virtual[1].URL)
	}
}
