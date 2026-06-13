package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RenderResolved reads the user config at path, deep-merges the pack layers
// beneath it, and returns the fully resolved config as YAML annotated with
// per-key provenance comments (# user or # pack: name). Output is
// deterministic — keys are sorted. This backs the `fuego config` command.
func RenderResolved(path string, packLayers []Layer) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	userMap := map[string]any{}
	if err := yaml.Unmarshal(data, &userMap); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	layers := make([]Layer, 0, len(packLayers)+1)
	layers = append(layers, packLayers...)
	layers = append(layers, Layer{Source: "user", Data: userMap})
	merged, prov := mergeLayers(layers)

	root := mapToYAMLNode(merged, prov, "")
	out, err := yaml.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("rendering resolved config: %w", err)
	}
	return out, nil
}

// mapToYAMLNode builds a YAML mapping node from m, attaching a provenance
// line comment to each key that has a recorded source. Nested maps recurse;
// a key with no recorded source (a subtree whose children come from mixed
// layers) is left uncommented.
func mapToYAMLNode(m map[string]any, prov *Provenance, prefix string) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range sortedMapKeys(m) {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: k}
		if comment := provComment(prov.Source(path)); comment != "" {
			keyNode.LineComment = comment
		}

		var valNode *yaml.Node
		if sub, ok := asStringMap(m[k]); ok {
			valNode = mapToYAMLNode(sub, prov, path)
		} else {
			valNode = &yaml.Node{}
			_ = valNode.Encode(m[k])
		}

		node.Content = append(node.Content, keyNode, valNode)
	}
	return node
}

func provComment(source string) string {
	switch source {
	case "":
		return ""
	case "user":
		return "user"
	default:
		return "pack: " + source
	}
}
