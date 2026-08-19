package main

import (
	"os"

	"github.com/tupora/ghostfleet/internal/bootstrap"
)

func main() {
	os.Exit(bootstrap.Run("ghostcli", os.Args[1:], os.Stdout, os.Stderr))
}
