package config

import "sort"

// Layer is one config source for merging, identified by a source name.
// The user's config.yaml uses the name "user"; packs use their pack name.
type Layer struct {
	Source string
	Data   map[string]any
}

// Provenance records which layer last set each dotted key path in the merged
// config, so `fuego config` can explain where every value came from.
type Provenance struct {
	sources map[string]string
}

func newProvenance() *Provenance {
	return &Provenance{sources: make(map[string]string)}
}

// Source returns the layer that set the value at the given dotted path, or
// "" if the path was never written (e.g. an engine default).
func (p *Provenance) Source(path string) string {
	if p == nil {
		return ""
	}
	return p.sources[path]
}

func (p *Provenance) set(path, source string) {
	p.sources[path] = source
}

// mergeLayers deep-merges layers in order (lowest precedence first; the user
// layer must be last) and returns the merged map plus provenance. Merge
// rules: maps merge key-wise, scalars and lists replace whole, the latest
// layer wins.
func mergeLayers(layers []Layer) (map[string]any, *Provenance) {
	merged := make(map[string]any)
	prov := newProvenance()
	for _, layer := range layers {
		mergeMap(merged, layer.Data, layer.Source, "", prov)
	}
	return merged, prov
}

func mergeMap(dst, src map[string]any, source, prefix string, prov *Provenance) {
	for _, k := range sortedMapKeys(src) {
		sv := src[k]
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}

		if srcMap, ok := asStringMap(sv); ok {
			dstMap, ok := dst[k].(map[string]any)
			if !ok {
				// dst has no map here (absent, or a scalar being replaced):
				// build a fresh map and record provenance for the subtree root.
				dstMap = make(map[string]any)
				dst[k] = dstMap
				prov.set(path, source)
			}
			mergeMap(dstMap, srcMap, source, path, prov)
			continue
		}

		// Scalar or list: replace whole.
		dst[k] = sv
		prov.set(path, source)
	}
}

// asStringMap normalizes a YAML-decoded mapping to map[string]any. yaml.v3
// decodes untyped mappings as map[string]any, but be defensive about
// map[any]any too.
func asStringMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = val
		}
		return out, true
	}
	return nil, false
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
