package main

import (
	"os"
	"github.com/Ecook14/gocrewwai/internal/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
