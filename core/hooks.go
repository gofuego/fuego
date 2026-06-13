package core

// AfterParseHook is called after PARSE, before ROUTE.
// It receives all parsed pages and returns the (possibly filtered/enriched) set.
type AfterParseHook func(pages []*Page) ([]*Page, error)

// IndexHook is called during INDEX, after taxonomy and collection virtual
// pages are generated but before the collision re-check. It is the supported
// way to add virtual pages: pages returned (with their URL set) flow through
// the same collision detection as engine-generated virtual pages.
type IndexHook func(pages []*Page) ([]*Page, error)

// BeforeRenderHook is called after INDEX, before RENDER.
// It receives all pages (including virtual taxonomy/collection pages)
// and returns the (possibly filtered/enriched) set.
type BeforeRenderHook func(pages []*Page) ([]*Page, error)

// Hooks bundles all registered pipeline hooks.
type Hooks struct {
	AfterParse   []AfterParseHook
	Index        []IndexHook
	BeforeRender []BeforeRenderHook
}
