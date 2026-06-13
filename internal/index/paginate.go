package index

import (
	"fmt"
	"strings"

	"github.com/FabioSol/fuego/core"
)

// paginate splits a listing virtual page into N pages of at most pageSize
// entry nodes each. The original page keeps its URL and serves page 1;
// pages 2..N live at {url}/page/{n}/. Every returned page carries a
// Paginator. With pageSize <= 0 or few enough entries, the page is
// returned untouched with a nil Paginator.
func paginate(page *core.Page, pageSize int) []*core.Page {
	nodes := page.Nodes
	if pageSize <= 0 || len(nodes) <= pageSize {
		return []*core.Page{page}
	}

	total := (len(nodes) + pageSize - 1) / pageSize
	base := strings.TrimSuffix(page.URL, "/")
	pageURL := func(n int) string {
		if n == 1 {
			return page.URL
		}
		return fmt.Sprintf("%s/page/%d/", base, n)
	}

	out := make([]*core.Page, 0, total)
	for n := 1; n <= total; n++ {
		start := (n - 1) * pageSize
		end := start + pageSize
		if end > len(nodes) {
			end = len(nodes)
		}

		current := page
		if n > 1 {
			clone := *page
			clone.RelPath = fmt.Sprintf("%s/page-%d", page.RelPath, n)
			clone.URL = pageURL(n)
			envelope := make(core.Envelope, len(page.Envelope))
			for k, v := range page.Envelope {
				envelope[k] = v
			}
			clone.Envelope = envelope
			current = &clone
		}

		current.Nodes = nodes[start:end]
		current.Paginator = &core.Paginator{
			CurrentPage: n,
			TotalPages:  total,
		}
		if n > 1 {
			current.Paginator.PrevURL = pageURL(n - 1)
		}
		if n < total {
			current.Paginator.NextURL = pageURL(n + 1)
		}

		out = append(out, current)
	}
	return out
}
