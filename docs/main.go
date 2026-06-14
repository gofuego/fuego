package main

import (
	"fmt"
	"os"

	"github.com/gofuego/fuego/docs/docspack"
	"github.com/gofuego/fuego/engine"
	"github.com/gofuego/fuego/parsers/markdown"
)

func main() {
	eng := engine.New()
	eng.Register(markdown.Parser())

	// Dogfood the pack system: the docs pack supplies the tags taxonomy and
	// the paginated tutorials collection as config defaults.
	eng.Use(docspack.Pack())

	if err := eng.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "fuego: %v\n", err)
		os.Exit(1)
	}
}
