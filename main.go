package main

import (
	"os"

	"github.com/ratelworks/agentlock/internal/agentlock"
)

func main() {
	os.Exit(agentlock.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
