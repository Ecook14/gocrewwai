package main

import (
	"fmt"
	"os"

	"github.com/Ecook14/crewai-go/internal/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
