package buildcache

import "github.com/gofuego/fuego/core"

// ClonePage returns a ParsedPage whose envelope and nodes share no mutable
// state with the input. The cache stores post-PARSE state; hooks mutate live
// pages in place, so both cache boundaries (snapshot and restore) must copy —
// otherwise hook output leaks into the cache and stale hook output leaks back
// out of it on a hit.
func ClonePage(pp ParsedPage) ParsedPage {
	pp.Envelope = CloneEnvelope(pp.Envelope)
	pp.Nodes = CloneNodes(pp.Nodes)
	if pp.Tree != nil {
		// Deep-copy every child so the isolation contract holds for every page
		// of a tree, not just the root.
		tree := make([]TreeNode, len(pp.Tree))
		for i, tn := range pp.Tree {
			tn.Envelope = CloneEnvelope(tn.Envelope)
			tn.Nodes = CloneNodes(tn.Nodes)
			tree[i] = tn
		}
		pp.Tree = tree
	}
	return pp
}

// CloneEnvelope deep-copies an envelope's JSON/YAML-shaped containers.
func CloneEnvelope(env core.Envelope) core.Envelope {
	if env == nil {
		return nil
	}
	return cloneValue(map[string]any(env)).(map[string]any)
}

// CloneNodes deep-copies a node tree (attributes and children; string content
// is immutable and shared).
func CloneNodes(nodes []core.Node) []core.Node {
	if nodes == nil {
		return nil
	}
	out := make([]core.Node, len(nodes))
	for i, n := range nodes {
		out[i] = n
		if n.Attributes != nil {
			out[i].Attributes = cloneValue(n.Attributes).(map[string]any)
		}
		out[i].Children = CloneNodes(n.Children)
	}
	return out
}

// cloneValue deep-copies the JSON/YAML-shaped containers an envelope may hold.
// Values of any other concrete type are returned as-is (shared): they sit
// outside the cacheable-envelope contract, and Save's per-page degradation
// keeps them from poisoning the cache.
func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		m := make(map[string]any, len(t))
		for k, val := range t {
			m[k] = cloneValue(val)
		}
		return m
	case []any:
		s := make([]any, len(t))
		for i, val := range t {
			s[i] = cloneValue(val)
		}
		return s
	case []map[string]any:
		s := make([]map[string]any, len(t))
		for i, val := range t {
			s[i] = cloneValue(val).(map[string]any)
		}
		return s
	case map[string]string:
		m := make(map[string]string, len(t))
		for k, val := range t {
			m[k] = val
		}
		return m
	case []map[string]string:
		s := make([]map[string]string, len(t))
		for i, val := range t {
			s[i] = cloneValue(val).(map[string]string)
		}
		return s
	case []string:
		return append([]string(nil), t...)
	case []int:
		return append([]int(nil), t...)
	case []float64:
		return append([]float64(nil), t...)
	case []bool:
		return append([]bool(nil), t...)
	default:
		return v
	}
}
