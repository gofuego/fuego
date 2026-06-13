---
title: Custom Parsers
layout: doc
nav_section: "Reference"
nav_weight: 4
tags:
  - reference
  - parsers
---

Fuego supports two ways to define content parsers: declarative (config-only) and compiled (Go code). Both produce the same universal AST.

## Universal AST

All parsers produce `[]Node`:

```go
type Node struct {
    Type       string
    Attributes map[string]any
    Content    string
    Children   []Node
    Raw        bool // pass Content through as raw HTML, unwrapped and unescaped
}
```

The engine never interprets node types. Your templates decide how `question`, `front`, `answer`, or any other type renders.

## Declarative Parsers

Define parsers with ordered regex rules in `config.yaml`. No Go code needed:

```yaml
parsers:
  trivia:
    rules:
      - match: '^\?\s*(.+)$'
        emit:
          type: question
          content: "$1"
      - match: '^\[([A-Z])\]\s+(.+)$'
        emit:
          type: answer
          content: "$2"
          attributes:
            letter: "$1"
```

This parses `.trivia` files line-by-line. First matching rule wins per line. Capture groups (`$1`, `$2`) are substituted into content and attributes.

## Compiled Parsers

Implement the `core.Parser` interface for full control. The parser receives the entire raw file and owns envelope extraction:

```go
type Parser interface {
    Type() string
    Parse(raw []byte) (Envelope, []Node, error)
}
```

Use `core.SplitFrontmatter(raw)` if your format carries YAML frontmatter, or the convenience wrappers `core.WithYAMLFrontmatter(...)` / `core.WithNoEnvelope(...)` for the common cases. For extensionless files like `Dockerfile`, also implement `core.FilenameParser` with `Filenames() []string`.

Register it in `main.go`:

```go
eng := engine.New()
eng.Register(&MyCustomParser{})
eng.Run(os.Args)
```

## Error Positions

Return a `core.ParseError` to point build errors at a source line:

```go
return nil, nil, &core.ParseError{
    Line: lineNo,
    Err:  fmt.Errorf("unexpected token %q", tok),
}
```

The build output then reads `[PARSE] error content/quiz.trivia:7: ...`. Reporting positions is optional — plain errors keep working. Frontmatter errors from `core.SplitFrontmatter` carry file-relative line numbers automatically.

## Parser Precedence

There are no built-in parsers — the engine ships with zero format opinions. When multiple parsers exist for the same file extension:

1. **Compiled** (registered via `eng.Register()`) — wins
2. **Declarative** (defined in `config.yaml`)

A file is discovered as content only if a registered parser matches its extension or filename.

## Markdown (opt-in)

Markdown is a first-party, opt-in parser — not a built-in. Register it like any other compiled parser:

```go
import "github.com/FabioSol/fuego/parsers/markdown"

eng.Register(markdown.Parser())
```

It uses goldmark with GitHub-Flavored Markdown (tables, strikethrough, autolinks, task lists) and emits a single node with `Raw: true` containing the rendered HTML.
