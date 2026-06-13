package render

import (
	"html/template"
	"text/template/parse"
)

// templateReferencesJSON reports whether the template or any of its
// associated templates (blocks, defines) reference .JSON (or $.JSON).
// Detection is conservative: any field/variable chain mentioning JSON
// counts, so over-detection computes an unused payload but never the
// reverse.
func templateReferencesJSON(t *template.Template) bool {
	for _, assoc := range t.Templates() {
		if assoc.Tree != nil && nodeUsesJSON(assoc.Tree.Root) {
			return true
		}
	}
	return false
}

func nodeUsesJSON(node parse.Node) bool {
	switch n := node.(type) {
	case *parse.ListNode:
		if n == nil {
			return false
		}
		for _, child := range n.Nodes {
			if nodeUsesJSON(child) {
				return true
			}
		}
	case *parse.ActionNode:
		return pipeUsesJSON(n.Pipe)
	case *parse.IfNode:
		return pipeUsesJSON(n.Pipe) || nodeUsesJSON(n.List) || nodeUsesJSON(n.ElseList)
	case *parse.RangeNode:
		return pipeUsesJSON(n.Pipe) || nodeUsesJSON(n.List) || nodeUsesJSON(n.ElseList)
	case *parse.WithNode:
		return pipeUsesJSON(n.Pipe) || nodeUsesJSON(n.List) || nodeUsesJSON(n.ElseList)
	case *parse.TemplateNode:
		return pipeUsesJSON(n.Pipe)
	case *parse.FieldNode:
		return identsContainJSON(n.Ident)
	case *parse.VariableNode:
		return identsContainJSON(n.Ident)
	case *parse.ChainNode:
		return identsContainJSON(n.Field) || nodeUsesJSON(n.Node)
	}
	return false
}

func pipeUsesJSON(pipe *parse.PipeNode) bool {
	if pipe == nil {
		return false
	}
	for _, cmd := range pipe.Cmds {
		for _, arg := range cmd.Args {
			if nodeUsesJSON(arg) {
				return true
			}
		}
	}
	return false
}

func identsContainJSON(idents []string) bool {
	for _, ident := range idents {
		if ident == "JSON" {
			return true
		}
	}
	return false
}
