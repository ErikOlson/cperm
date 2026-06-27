package main

import (
	"os"

	"github.com/erikmav/cperm/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
