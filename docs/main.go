package main

import (
	"fmt"
	"os"

	"github.com/FabioSol/fuego/engine"
	"github.com/FabioSol/fuego/parsers/markdown"
)

func main() {
	eng := engine.New()
	eng.Register(markdown.Parser())

	if err := eng.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "fuego: %v\n", err)
		os.Exit(1)
	}
}
