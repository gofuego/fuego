---
title: Custom Parsers
layout: doc
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

Implement the `core.Parser` interface for full control:

```go
type Parser interface {
    Type() string
    Parse(rawPayload []byte, meta Envelope) ([]Node, error)
}
```

Register it in `main.go`:

```go
eng := engine.New()
eng.Register(&MyCustomParser{})
eng.Run(os.Args)
```

## Parser Precedence

When multiple parsers exist for the same file extension:

1. **Compiled** (registered via `eng.Register()`) — highest priority
2. **Declarative** (defined in `config.yaml`) — middle priority
3. **Built-in** (Markdown) — lowest priority

This means you can override the built-in Markdown parser by registering your own, or define a declarative one in config.

## Built-in Markdown

`.md` files are parsed automatically using goldmark with GitHub-Flavored Markdown (tables, strikethrough, autolinks, task lists). The output is a single `Node{Type: "html"}` containing rendered HTML.
