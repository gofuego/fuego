package core

// AfterParseHook is called after PARSE, before ROUTE.
// It receives all parsed pages and returns the (possibly filtered/enriched) set.
type AfterParseHook func(pages []*Page) ([]*Page, error)

// BeforeRenderHook is called after INDEX, before RENDER.
// It receives all pages (including virtual taxonomy/collection pages)
// and returns the (possibly filtered/enriched) set.
type BeforeRenderHook func(pages []*Page) ([]*Page, error)

// Hooks bundles all registered pipeline hooks.
type Hooks struct {
	AfterParse   []AfterParseHook
	BeforeRender []BeforeRenderHook
}
