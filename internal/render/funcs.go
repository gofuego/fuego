package render

import (
	"fmt"
	"html/template"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/FabioSol/fuego/core"
)

// buildFuncMap constructs the function map shared by the base template,
// layouts, renderers, and partials.
func (tc *TemplateCache) buildFuncMap() template.FuncMap {
	return template.FuncMap{
		"render": func(nodes []core.Node) template.HTML {
			return tc.renderWithOverrides(nodes)
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"partial":    tc.execPartial,
		"dict":       dictFunc,
		"where":      whereFunc,
		"sortBy":     sortByFunc,
		"limit":      limitFunc,
		"first":      firstFunc,
		"dateFormat": dateFormatFunc,
	}
}

// execPartial executes a template from theme/partials/ by name. The optional
// second argument is passed as the partial's data (nil if omitted).
func (tc *TemplateCache) execPartial(name string, data ...any) (template.HTML, error) {
	tmpl, ok := tc.partials[name]
	if !ok {
		available := make([]string, 0, len(tc.partials))
		for n := range tc.partials {
			available = append(available, n)
		}
		sort.Strings(available)
		if len(available) == 0 {
			return "", fmt.Errorf("partial %q not found: theme/partials/ has no templates", name)
		}
		return "", fmt.Errorf("partial %q not found in theme/partials/ (available: %s)",
			name, strings.Join(available, ", "))
	}

	var arg any
	if len(data) > 0 {
		arg = data[0]
	}

	var sb strings.Builder
	if err := tmpl.Execute(&sb, arg); err != nil {
		return "", fmt.Errorf("executing partial %q: %w", name, err)
	}
	return template.HTML(sb.String()), nil
}

// dictFunc builds a map from alternating key/value arguments, for passing
// multiple values to a partial: {{partial "footer" (dict "year" "2026")}}.
func dictFunc(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments (%d)", len(pairs))
	}
	m := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key %v (position %d) is not a string", pairs[i], i)
		}
		m[key] = pairs[i+1]
	}
	return m, nil
}

// whereFunc filters a slice, keeping elements whose value for key equals
// value. Keys resolve against maps, struct fields (case-insensitive), or a
// struct's Envelope map. Comparison is loose: values match if deeply equal
// or if their string forms are equal, so envelope numbers and YAML scalars
// compare predictably.
func whereFunc(collection any, key string, value any) (any, error) {
	v, err := sliceValue("where", collection)
	if err != nil {
		return nil, err
	}
	out := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		got, ok := lookupKey(elem, key)
		if ok && looseEqual(got, value) {
			out = reflect.Append(out, elem)
		}
	}
	return out.Interface(), nil
}

// sortByFunc returns a copy of the slice sorted by the value at key.
// An optional trailing "desc" reverses the order. Numeric values sort
// numerically, everything else by string form; elements missing the key
// sort first.
func sortByFunc(collection any, key string, order ...string) (any, error) {
	v, err := sliceValue("sortBy", collection)
	if err != nil {
		return nil, err
	}

	desc := false
	if len(order) > 0 {
		switch order[0] {
		case "asc":
		case "desc":
			desc = true
		default:
			return nil, fmt.Errorf("sortBy: order must be \"asc\" or \"desc\", got %q", order[0])
		}
	}

	// Copy so shared slices (e.g. .Site.Pages) are never mutated.
	out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
	reflect.Copy(out, v)

	sort.SliceStable(out.Interface(), func(i, j int) bool {
		a, _ := lookupKey(out.Index(i), key)
		b, _ := lookupKey(out.Index(j), key)
		less := lessValues(a, b)
		if desc {
			return lessValues(b, a)
		}
		return less
	})

	return out.Interface(), nil
}

// limitFunc returns at most n leading elements. The collection is the last
// argument so it composes in pipelines: {{range .Site.Pages | limit 5}}.
func limitFunc(n int, collection any) (any, error) {
	v, err := sliceValue("limit", collection)
	if err != nil {
		return nil, err
	}
	if n < 0 {
		n = 0
	}
	if n > v.Len() {
		n = v.Len()
	}
	return v.Slice(0, n).Interface(), nil
}

// firstFunc returns the first element of a slice, or nil if it is empty.
func firstFunc(collection any) (any, error) {
	v, err := sliceValue("first", collection)
	if err != nil {
		return nil, err
	}
	if v.Len() == 0 {
		return nil, nil
	}
	return v.Index(0).Interface(), nil
}

// dateFormatFunc formats a time.Time or a date string using a Go reference
// layout: {{dateFormat "Jan 2, 2006" .Page.Envelope.date}}.
func dateFormatFunc(layout string, value any) (string, error) {
	t, err := toTime(value)
	if err != nil {
		return "", err
	}
	return t.Format(layout), nil
}

var dateInputFormats = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func toTime(value any) (time.Time, error) {
	switch v := value.(type) {
	case time.Time:
		return v, nil
	case string:
		for _, format := range dateInputFormats {
			if t, err := time.Parse(format, v); err == nil {
				return t, nil
			}
		}
		return time.Time{}, fmt.Errorf("dateFormat: cannot parse %q (accepted: RFC3339, 2006-01-02T15:04:05, 2006-01-02 15:04:05, 2006-01-02)", v)
	default:
		return time.Time{}, fmt.Errorf("dateFormat: expected time.Time or string, got %T", value)
	}
}

func sliceValue(funcName string, collection any) (reflect.Value, error) {
	v := reflect.ValueOf(collection)
	for v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if !v.IsValid() || (v.Kind() != reflect.Slice && v.Kind() != reflect.Array) {
		return reflect.Value{}, fmt.Errorf("%s: expected a slice, got %T", funcName, collection)
	}
	return v, nil
}

// lookupKey resolves key against an element: map index for maps, exported
// field (case-insensitive) for structs, then the struct's Envelope map.
func lookupKey(elem reflect.Value, key string) (any, bool) {
	for elem.Kind() == reflect.Interface || elem.Kind() == reflect.Pointer {
		if elem.IsNil() {
			return nil, false
		}
		elem = elem.Elem()
	}

	switch elem.Kind() {
	case reflect.Map:
		if elem.Type().Key().Kind() != reflect.String {
			return nil, false
		}
		got := elem.MapIndex(reflect.ValueOf(key))
		if !got.IsValid() {
			return nil, false
		}
		return got.Interface(), true

	case reflect.Struct:
		field := elem.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(name, key)
		})
		if field.IsValid() && field.CanInterface() {
			return field.Interface(), true
		}
		envelope := elem.FieldByName("Envelope")
		if envelope.IsValid() && envelope.Kind() == reflect.Map {
			return lookupKey(envelope, key)
		}
	}
	return nil, false
}

func looseEqual(a, b any) bool {
	if reflect.DeepEqual(a, b) {
		return true
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func lessValues(a, b any) bool {
	af, aNum := toFloat(a)
	bf, bNum := toFloat(b)
	if aNum && bNum {
		return af < bf
	}
	return stringify(a) < stringify(b)
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
