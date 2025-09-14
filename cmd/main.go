package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
)

type ScopeInsert struct {
	TargetName string
	ScopeURLs  []string
}

type CLIFlags struct {
	Target       string
	Stats        bool
	InsertScope  ScopeInsert
	InsertTarget string
}

func ParseFlags() (CLIFlags, error) {
	var target string
	var stats bool
	var insertScope string
	var insertTarget string

	flag.StringVar(&target, "t", "", "Show target stats based on target name (<target_name>")
	flag.BoolVar(&stats, "s", false, "Show stats")
	flag.StringVar(&insertScope, "iS", "", "Comma-separated values for scope. First value should be target name and the following values will be interpreted as scope URLs (<target_name>,<scope_url>")
	flag.StringVar(&insertTarget, "iT", "", "Insert target (<target_name>")

	flag.Parse()

	if target == "" && stats == false && insertScope == "" && insertTarget == "" {
		return CLIFlags{}, fmt.Errorf("Error running target-tracker CLI: At lest one flag must be present")
	}

	if target != "" && stats == true {
		return CLIFlags{}, fmt.Errorf("Error running target-tracekr CLI: -t and -s flag are mutually exclusive")
	}

	scope := ScopeInsert{}
	if insertScope != "" {
		scopeURLs := make([]string, 0)

		args := strings.Split(insertScope, ",")
		for i, arg := range args {
			if i == 0 {
				scope.TargetName = arg
				continue
			}

			_, err := url.Parse(arg)
			if err != nil {
				return CLIFlags{}, fmt.Errorf("Invalid scope URL: %s", arg)
			}

			scopeURLs = append(scopeURLs, arg)
		}

		scope.ScopeURLs = scopeURLs
	}

	return CLIFlags{
		Target:       target,
		Stats:        stats,
		InsertScope:  scope,
		InsertTarget: insertTarget,
	}, nil
}

func main() {
	flags, err := ParseFlags()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if flags.Stats {
		// TODO: Show stats
		return
	}

	if flags.InsertTarget != "" {
		if err := InsertTarget(flags.InsertTarget); err != nil {
			fmt.Printf("Error inserting target: %s\n", err.Error())
			os.Exit(1)
		}
	}

	if flags.InsertScope.TargetName != "" {
		if err := InsertScope(flags.InsertScope); err != nil {
			fmt.Printf("Error inserting scope: %s\n", err.Error())
			os.Exit(1)
		}
	}

	if flags.Target != "" {
		cli, err := NewCLI(flags.Target)
		if err != nil {
			fmt.Printf("Error running target-tracekr CLI: %s\n", err.Error())
			os.Exit(1)
		}

		if err := cli.RenderURLsTable(); err != nil {
			log.Fatalf("Failed to render table: %s", err.Error())
		}
	}
}
