package core

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

var fmDelimiter = []byte("---")

var yamlLineRe = regexp.MustCompile(`yaml: line (\d+):`)

// yamlErrorFileLine maps a yaml.v3 error to a 1-based line in the original
// file. YAML errors report lines relative to the frontmatter block, whose
// first content line sits at file line firstYAMLLine.
func yamlErrorFileLine(err error, firstYAMLLine int) int {
	m := yamlLineRe.FindStringSubmatch(err.Error())
	if m == nil {
		return 0
	}
	n, convErr := strconv.Atoi(m[1])
	if convErr != nil {
		return 0
	}
	return firstYAMLLine + n - 1
}

// SplitFrontmatter separates a content file's raw bytes into the YAML
// frontmatter envelope and the remaining payload. The frontmatter must
// be enclosed between two "---" lines at the start of the file.
//
// If no frontmatter is found, an empty Envelope is returned and the
// entire content is treated as the payload.
func SplitFrontmatter(raw []byte) (Envelope, []byte, error) {
	trimmed := bytes.TrimLeft(raw, " \t\r\n")

	if !bytes.HasPrefix(trimmed, fmDelimiter) {
		return Envelope{}, raw, nil
	}

	// 1-based file line of the opening --- (leading blank lines shift it).
	openLine := 1 + bytes.Count(raw[:len(raw)-len(trimmed)], []byte("\n"))

	// Find the closing delimiter
	rest := trimmed[len(fmDelimiter):]
	// Skip the newline after opening ---
	if idx := bytes.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		return Envelope{}, raw, nil
	}

	closeIdx := bytes.Index(rest, fmDelimiter)
	if closeIdx < 0 {
		return nil, nil, &ParseError{
			Line: openLine,
			Err:  fmt.Errorf("unclosed frontmatter: missing closing ---"),
		}
	}

	frontmatterBytes := rest[:closeIdx]
	payload := rest[closeIdx+len(fmDelimiter):]

	// Strip leading newline from payload
	if len(payload) > 0 && payload[0] == '\n' {
		payload = payload[1:]
	} else if len(payload) > 1 && payload[0] == '\r' && payload[1] == '\n' {
		payload = payload[2:]
	}

	envelope := make(Envelope)
	if err := yaml.Unmarshal(frontmatterBytes, &envelope); err != nil {
		return nil, nil, &ParseError{
			Line: yamlErrorFileLine(err, openLine+1),
			Err:  fmt.Errorf("parsing frontmatter YAML: %w", err),
		}
	}

	return envelope, payload, nil
}
