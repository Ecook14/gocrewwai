package main

import (
	"github.com/Ecook14/gocrewwai/internal/cli"
	"os"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
