// Package core defines the fundamental types shared across the Fuego engine.
// Both the public engine package and internal packages import from here,
// avoiding import cycles.
package core

// Envelope encapsulates the metadata parsed from content file frontmatter.
type Envelope = map[string]any

// Node represents a single element in the universal AST tree.
type Node struct {
	Type       string         `json:"type"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Content    string         `json:"content,omitempty"`
	Children   []Node         `json:"children,omitempty"`
	Raw        bool           `json:"raw,omitempty"`
}

// Parser defines the interface for content parsers.
// Parsers receive the entire raw file and return both the extracted
// envelope (metadata) and the parsed AST nodes.
type Parser interface {
	Type() string
	Parse(raw []byte) (Envelope, []Node, error)
}

// FilenameParser is an optional interface that parsers can implement
// to declare filename patterns they handle. This allows parsers to
// process extensionless files like Dockerfile or Makefile.
// Patterns are matched against the base filename (not the full path).
type FilenameParser interface {
	Parser
	Filenames() []string
}

// PageTree is the recursive shape a TreeParser returns: a root node
// (Envelope + Nodes) plus a set of Children keyed by relative slug path,
// nested arbitrarily. It lets one source artifact (an OpenAPI spec, a DBML
// schema) expand into a whole section of real pages — an index plus a tree
// of pages per operation, table, or suite — while the engine stays agnostic
// about what the format means.
//
// The root PageTree's Envelope and Nodes become the routed root page (the
// source file's own page). Each entry in Children becomes a real core.Page:
//
//   - its RelPath is the source file's RelPath joined with the child's slug
//     path (so identity is source-file + slug-path, never colliding with a
//     plain page unless intended);
//   - its URL is the root's routed URL joined with the child's slug-path
//     segments — composed AFTER the root goes through the normal three-tier
//     routing, so index-file and route-pattern conventions on the root are
//     honored;
//   - its Envelope and Nodes are the child node's own, so children flow
//     through INDEX and are seen natively by taxonomies, collections, and
//     pagination.
//
// A Children key is a relative slug path (one or more "/"-separated
// segments); nesting a PageTree under a key extends that path. Two children
// whose slug paths compose to the same URL — a duplicate sibling slug, or a
// child colliding with another page — surface through the engine's existing
// ROUTE/INDEX collision detection (a GlobalFatal), not through any tree-local
// check. Envelope values should be JSON-shaped so tree pages stay
// cache-eligible; a missing per-child layout falls back to the base template.
type PageTree struct {
	Envelope Envelope
	Nodes    []Node
	Children map[string]*PageTree
}

// TreeParser is an optional interface a Parser may also implement to expand
// one artifact into a tree of real pages. The engine detects it by interface
// assertion at PARSE — there is no registration change and plain Parsers are
// untouched. When a parser implements TreeParser, the engine calls ParseTree
// (Parse is not called for that file) and expands the returned PageTree.
type TreeParser interface {
	Parser
	ParseTree(raw []byte) (*PageTree, error)
}

// Page is the central data carrier flowing through the pipeline.
// It is progressively enriched by each phase: DISCOVER sets paths,
// PARSE sets envelope/nodes, ROUTE sets URL, INDEX may annotate further.
type Page struct {
	SourcePath string   // absolute path to the source file
	RelPath    string   // path relative to content dir
	Ext        string   // file extension without dot
	Envelope   Envelope // parsed frontmatter metadata
	Nodes      []Node   // parsed AST body
	URL        string   // resolved output URL (set by route phase)
	Layout     string   // layout name from envelope
	Type       string   // content type (from envelope or extension)
	IsRaw      bool     // true if no parser matched (raw passthrough)
	Skip       bool     // exclude from RENDER and the manifest (drafts, pack-internal pages)
	Paginator  *Paginator // set on paginated listing pages, nil otherwise

	// TreeRootRel and TreeSlugPath are set only on pages expanded from a
	// TreeParser's PageTree (see PageTree). TreeRootRel is the RelPath of the
	// tree's routed root page; TreeSlugPath is this child's slug path relative
	// to that root (e.g. "tags/billing/get-invoice"). ROUTE composes the
	// child's URL as the root's resolved URL joined with the slug-path
	// segments, so the root's three-tier routing (including the index-file
	// convention) is honored before children are placed under it. Both are
	// empty on ordinary pages and on tree roots.
	TreeRootRel  string
	TreeSlugPath string
}

// Paginator describes a page's position in a paginated listing. The INDEX
// phase sets it on taxonomy term and collection pages split by page_size;
// templates reach it as .Paginator.
type Paginator struct {
	CurrentPage int
	TotalPages  int
	PrevURL     string // empty on the first page
	NextURL     string // empty on the last page
}
