package render

import (
	"encoding/json"
	"fmt"
	"html/template"
	"strings"

	"github.com/gofuego/fuego/core"
)

// DefaultRenderer produces semantic HTML from a []Node tree using
// <div data-type="..."> wrappers. This is the fallback when no
// per-type renderer template is available.
func DefaultRenderer(nodes []core.Node) template.HTML {
	var sb strings.Builder
	renderNodes(&sb, nodes)
	return template.HTML(sb.String())
}

func renderNodes(sb *strings.Builder, nodes []core.Node) {
	for _, n := range nodes {
		renderNode(sb, n)
	}
}

func renderNode(sb *strings.Builder, n core.Node) {
	// Nodes marked Raw contain pre-rendered HTML (e.g., from the Markdown parser).
	// Output directly without wrapping or escaping.
	if n.Raw {
		sb.WriteString(n.Content)
		return
	}

	sb.WriteString(`<div data-type="`)
	sb.WriteString(template.HTMLEscapeString(n.Type))
	sb.WriteString(`"`)

	if len(n.Attributes) > 0 {
		attrsJSON, err := json.Marshal(n.Attributes)
		if err == nil {
			sb.WriteString(` data-attrs="`)
			sb.WriteString(template.HTMLEscapeString(string(attrsJSON)))
			sb.WriteString(`"`)
		}
	}

	sb.WriteString(`>`)

	if n.Content != "" {
		sb.WriteString(template.HTMLEscapeString(n.Content))
	}

	if len(n.Children) > 0 {
		renderNodes(sb, n.Children)
	}

	sb.WriteString(`</div>`)
}

// renderWithOverrides renders nodes using per-type renderer templates
// when available, falling back to the default semantic renderer.
func (tc *TemplateCache) renderWithOverrides(nodes []core.Node) template.HTML {
	var sb strings.Builder
	for _, n := range nodes {
		if tmpl, ok := tc.renderers[n.Type]; ok {
			tmpl.Execute(&sb, n)
		} else {
			renderNode(&sb, n)
		}
	}
	return template.HTML(sb.String())
}

// JSONPayload serializes the page envelope and nodes into a JSON string
// safe for embedding in HTML via <script type="application/json">.
// Uses json.Marshal which escapes <, >, & to unicode sequences.
func JSONPayload(envelope core.Envelope, nodes []core.Node, url string) (string, error) {
	payload := map[string]any{
		"envelope": envelope,
		"nodes":    nodes,
		"url":      url,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling page data: %w", err)
	}
	return string(data), nil
}
