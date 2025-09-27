package main

import (
	"flag"
	"fmt"
	"net/url"
	"strings"
)

type ScopeInsert struct {
	TargetName string
	ScopeURLs  []string
}

type CLIFlags struct {
	Target        string
	Stats         bool
	InsertScope   ScopeInsert
	InsertTarget  string
	DisableTarget string
}

func ParseFlags() (CLIFlags, error) {
	var target string
	var stats bool
	var insertScope string
	var insertTarget string
	var disableTarget string

	flag.StringVar(&target, "t", "", "Show target stats based on target name (<target_name>)")
	flag.BoolVar(&stats, "s", false, "Show stats")
	flag.StringVar(&insertScope, "iS", "", "Comma-separated values for scope. First value should be target name, and the following values will be interpreted as scope URLs (<target_name>,<true/false>,<scope_url>)")
	flag.StringVar(&insertTarget, "iT", "", "Insert target (<target_name>)")
	flag.StringVar(&disableTarget, "dT", "", "Disable target (<target_name>)")

	flag.Parse()

	if target == "" && stats == false && insertScope == "" && insertTarget == "" {
		return CLIFlags{}, fmt.Errorf("Error running ScopeWarden CLI: At lest one flag must be present")
	}

	if target != "" && stats == true {
		return CLIFlags{}, fmt.Errorf("Error running ScopeWarden CLI: -t and -s flag are mutually exclusive")
	}

	scope := ScopeInsert{}
	if insertScope != "" {
		scopeURLs := make([]string, 0)

		args := strings.Split(insertScope, ",")
		for i, arg := range args {
			// Check target name
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
		Target:        target,
		Stats:         stats,
		InsertScope:   scope,
		InsertTarget:  insertTarget,
		DisableTarget: disableTarget,
	}, nil
}
