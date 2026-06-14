package parse

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofuego/fuego/core"
	"github.com/gofuego/fuego/internal/config"
)

// compiledRule is a single regex rule with its pre-compiled pattern and emit template.
type compiledRule struct {
	pattern *regexp.Regexp
	emit    config.EmitConfig
}

// DeclarativeParser implements core.Parser using ordered regex rules from config.
// Each line of the payload is matched against rules top-to-bottom; first match wins.
// Capture group substitution ($0, $1, $2, ...) is applied to the emit content and attributes.
type DeclarativeParser struct {
	name  string
	rules []compiledRule
}

// NewDeclarativeParser compiles regex rules from config and returns a ready-to-use parser.
// Returns an error if any regex pattern fails to compile.
func NewDeclarativeParser(name string, cfg config.ParserConfig) (*DeclarativeParser, error) {
	rules := make([]compiledRule, 0, len(cfg.Rules))

	for i, rule := range cfg.Rules {
		re, err := regexp.Compile(rule.Match)
		if err != nil {
			return nil, fmt.Errorf("parser %q rule %d: invalid regex %q: %w", name, i, rule.Match, err)
		}
		rules = append(rules, compiledRule{
			pattern: re,
			emit:    rule.Emit,
		})
	}

	return &DeclarativeParser{name: name, rules: rules}, nil
}

func (p *DeclarativeParser) Type() string { return p.name }

func (p *DeclarativeParser) Parse(raw []byte) (core.Envelope, []core.Node, error) {
	env, payload, err := core.SplitFrontmatter(raw)
	if err != nil {
		return nil, nil, err
	}
	if env == nil {
		env = make(core.Envelope)
	}

	text := strings.TrimSpace(string(payload))
	if text == "" {
		return env, nil, nil
	}

	lines := strings.Split(text, "\n")
	var nodes []core.Node

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}

		node, matched := p.matchLine(line)
		if !matched {
			// Unmatched lines are silently skipped — they don't produce nodes
			continue
		}
		nodes = append(nodes, node)
	}

	return env, nodes, nil
}

// matchLine evaluates rules top-to-bottom against a line. First match wins.
func (p *DeclarativeParser) matchLine(line string) (core.Node, bool) {
	for _, rule := range p.rules {
		matches := rule.pattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		node := core.Node{
			Type:    substituteGroups(rule.emit.Type, matches),
			Content: substituteGroups(rule.emit.Content, matches),
		}

		if len(rule.emit.Attributes) > 0 {
			node.Attributes = make(map[string]any, len(rule.emit.Attributes))
			for k, v := range rule.emit.Attributes {
				if s, ok := v.(string); ok {
					node.Attributes[k] = substituteGroups(s, matches)
				} else {
					node.Attributes[k] = v
				}
			}
		}

		return node, true
	}
	return core.Node{}, false
}

// substituteGroups replaces $0, $1, $2, ... with their corresponding capture group values.
func substituteGroups(template string, matches []string) string {
	if !strings.Contains(template, "$") {
		return template
	}

	result := template
	for i, m := range matches {
		result = strings.ReplaceAll(result, fmt.Sprintf("$%d", i), m)
	}
	return result
}
