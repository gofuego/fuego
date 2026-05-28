package main

import (
	"fmt"
	"os"

	"github.com/FabioSol/fuego/engine"
)

func main() {
	eng := engine.New()

	if err := eng.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "fuego: %v\n", err)
		os.Exit(1)
	}
}
