package render

import (
	"html/template"
	"text/template/parse"
)

// A page's rendered output depends on other pages only when its template
// reads the site-wide page list, .Site.Pages. (.Site.Name and .Site.BaseURL
// are build-constants — a config change invalidates the whole cache anyway.)
// Detecting .Site.Pages usage — including through partials — lets an
// incremental build skip re-rendering content pages whose layout is
// "site-blind".

// treeFacts summarizes the parts of a template tree relevant to narrowing.
type treeFacts struct {
	sitePages  bool            // references .Site.Pages directly
	partials   map[string]bool // partial "name" calls with a literal name
	dynPartial bool            // a partial call with a non-literal name (conservative)
}

func templateFacts(t *template.Template) treeFacts {
	f := treeFacts{partials: map[string]bool{}}
	for _, assoc := range t.Templates() {
		if assoc.Tree != nil {
			walkSite(assoc.Tree.Root, &f)
		}
	}
	return f
}

func walkSite(node parse.Node, f *treeFacts) {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return
		}
		for _, c := range n.Nodes {
			walkSite(c, f)
		}
	case *parse.ActionNode:
		walkPipeSite(n.Pipe, f)
	case *parse.IfNode:
		walkPipeSite(n.Pipe, f)
		walkSite(n.List, f)
		walkSite(n.ElseList, f)
	case *parse.RangeNode:
		walkPipeSite(n.Pipe, f)
		walkSite(n.List, f)
		walkSite(n.ElseList, f)
	case *parse.WithNode:
		walkPipeSite(n.Pipe, f)
		walkSite(n.List, f)
		walkSite(n.ElseList, f)
	case *parse.TemplateNode:
		walkPipeSite(n.Pipe, f)
	}
}

func walkPipeSite(pipe *parse.PipeNode, f *treeFacts) {
	if pipe == nil {
		return
	}
	for _, cmd := range pipe.Cmds {
		if len(cmd.Args) > 0 {
			if id, ok := cmd.Args[0].(*parse.IdentifierNode); ok && id.Ident == "partial" {
				switch {
				case len(cmd.Args) < 2:
					f.dynPartial = true
				default:
					if s, ok := cmd.Args[1].(*parse.StringNode); ok {
						f.partials[s.Text] = true
					} else {
						f.dynPartial = true
					}
				}
			}
		}
		for _, arg := range cmd.Args {
			walkArgSite(arg, f)
		}
	}
}

func walkArgSite(arg parse.Node, f *treeFacts) {
	switch a := arg.(type) {
	case *parse.FieldNode:
		if identsHaveSitePages(a.Ident) {
			f.sitePages = true
		}
	case *parse.VariableNode:
		if identsHaveSitePages(a.Ident) {
			f.sitePages = true
		}
	case *parse.ChainNode:
		if identsHaveSitePages(a.Field) {
			f.sitePages = true
		}
		walkArgSite(a.Node, f)
	case *parse.PipeNode:
		walkPipeSite(a, f)
	}
}

// identsHaveSitePages reports whether a field chain contains "Site" directly
// followed by "Pages", matching .Site.Pages and $.Site.Pages.
func identsHaveSitePages(idents []string) bool {
	for i := 0; i+1 < len(idents); i++ {
		if idents[i] == "Site" && idents[i+1] == "Pages" {
			return true
		}
	}
	return false
}

// computeSitePages returns, for the base template and every layout, whether
// rendering it reads .Site.Pages — directly or via any partial it calls
// (transitively). partials maps partial name to its parsed template.
func computeSitePages(base *template.Template, layouts, partials map[string]*template.Template) (forBase bool, forLayouts map[string]bool, partialDeps map[string]bool) {
	// Facts per partial.
	pf := make(map[string]treeFacts, len(partials))
	for name, p := range partials {
		pf[name] = templateFacts(p)
	}

	// Fixpoint: a partial uses .Site.Pages if it references it directly, calls
	// a partial dynamically (conservative), or calls a partial that does.
	partialDeps = make(map[string]bool, len(partials))
	for name, f := range pf {
		if f.sitePages || f.dynPartial {
			partialDeps[name] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for name, f := range pf {
			if partialDeps[name] {
				continue
			}
			for called := range f.partials {
				if partialDeps[called] {
					partialDeps[name] = true
					changed = true
					break
				}
			}
		}
	}

	uses := func(t *template.Template) bool {
		f := templateFacts(t)
		if f.sitePages || f.dynPartial {
			return true
		}
		for called := range f.partials {
			if partialDeps[called] {
				return true
			}
		}
		return false
	}

	forBase = uses(base)
	forLayouts = make(map[string]bool, len(layouts))
	for name, l := range layouts {
		forLayouts[name] = uses(l)
	}
	return forBase, forLayouts, partialDeps
}
