package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	flags, err := ParseFlags()
	if err != nil {
		log.Fatal(err.Error())
		os.Exit(1)
	}

	cli, err := NewCLI()
	if err != nil {
		exitWithErr(fmt.Sprintf("Error running target-tracekr CLI: %s\n", err.Error()))
	}

	if flags.Stats {
		if err := cli.RenderStatsTable(); err != nil {
			exitWithErr(fmt.Sprintf("Failed to render stats table: %s", err.Error()))
		}

		return
	}

	if flags.DisableTarget != "" {
		if err := DisableTarget(flags.DisableTarget); err != nil {
			exitWithErr(fmt.Sprintf("Error disabing target: %s\n", err.Error()))
		}
	}

	if flags.InsertTarget != "" {
		if err := InsertTarget(flags.InsertTarget); err != nil {
			exitWithErr(fmt.Sprintf("Error inserting target: %s\n", err.Error()))
		}
	}

	if flags.InsertScope.TargetName != "" {
		if err := InsertScope(flags.InsertScope); err != nil {
			exitWithErr(fmt.Sprintf("Error inserting scope: %s\n", err.Error()))
		}
	}

	if flags.Target != "" {
		if err := cli.SetTarget(flags.Target); err != nil {
			exitWithErr(fmt.Sprintf("Failed to set target: %s", err.Error()))
		}

		if err := cli.RenderURLsTable(); err != nil {
			exitWithErr(fmt.Sprintf("Failed to render table: %s", err.Error()))
		}
	}
}

func exitWithErr(errMsg string) {
	log.Println(errMsg)
	os.Exit(1)
}
