package main

import (
	"os"

	"github.com/OysterD3/wtmux/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
