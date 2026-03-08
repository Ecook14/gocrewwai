package main

import (
	"os"
	"github.com/Ecook14/gocrew/internal/cli"
)

func main() {
	if err := cli.Run(os.Args); err != nil {
		os.Exit(1)
	}
}
