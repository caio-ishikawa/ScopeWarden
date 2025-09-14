package main

import (
	"log"
)

// TODO: Flags
func main() {
	cli, err := NewCLI("NASA")
	if err != nil {
		log.Fatal("Failed to start CLI: %w", err)
	}

	if err := cli.RenderURLsTable(); err != nil {
		log.Fatalf("Failed to render table: %s", err.Error())
	}
}
