package core

// ParseFunc is the signature for a payload-level parse function used
// with convenience wrappers like WithYAMLFrontmatter.
type ParseFunc func(payload []byte, meta Envelope) ([]Node, error)

// WithYAMLFrontmatter returns a Parser that splits YAML frontmatter
// (--- delimited) from the raw file, then delegates the remaining
// payload and parsed envelope to fn.
func WithYAMLFrontmatter(typeName string, fn ParseFunc) Parser {
	return &yamlFrontmatterParser{typeName: typeName, fn: fn}
}

type yamlFrontmatterParser struct {
	typeName string
	fn       ParseFunc
}

func (p *yamlFrontmatterParser) Type() string { return p.typeName }

func (p *yamlFrontmatterParser) Parse(raw []byte) (Envelope, []Node, error) {
	env, payload, err := SplitFrontmatter(raw)
	if err != nil {
		return nil, nil, err
	}
	if env == nil {
		env = make(Envelope)
	}
	nodes, err := p.fn(payload, env)
	if err != nil {
		return nil, nil, err
	}
	return env, nodes, nil
}

// RawParseFunc is the signature for a parser that handles raw bytes
// and produces both envelope and nodes.
type RawParseFunc func(raw []byte) (Envelope, []Node, error)

// WithNoEnvelope returns a Parser that passes the entire raw file to fn
// without any envelope extraction. The parser is responsible for
// producing both the envelope and the nodes from the raw content.
func WithNoEnvelope(typeName string, fn RawParseFunc) Parser {
	return &noEnvelopeParser{typeName: typeName, fn: fn}
}

type noEnvelopeParser struct {
	typeName string
	fn       RawParseFunc
}

func (p *noEnvelopeParser) Type() string { return p.typeName }

func (p *noEnvelopeParser) Parse(raw []byte) (Envelope, []Node, error) {
	return p.fn(raw)
}
