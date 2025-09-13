package main

import (
	"log"
)

func main() {
	cli, err := NewCLI()
	if err != nil {
		log.Fatal("Failed to start CLI: %w", err)
	}

	log.Println(cli)

	target, err := cli.db.GetTargetByName("NASA")
	if err != nil {
		log.Fatal("Failed to get domain by name: %w", err)
	}

	log.Println("Rendering table")
	if err := cli.RenderURLsTable(target.UUID); err != nil {
		log.Fatalf("Failed to render table: %s", err.Error())
	}
}
