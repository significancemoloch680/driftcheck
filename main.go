package main

import (
	"os"

	"github.com/ratelworks/driftcheck/internal/driftcheck"
)

func main() {
	os.Exit(driftcheck.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
