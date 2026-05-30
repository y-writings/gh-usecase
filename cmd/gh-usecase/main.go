package main

import (
	"os"

	"github.com/y-writings/gh-usecase/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
