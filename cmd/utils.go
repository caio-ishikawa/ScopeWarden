package main

import (
	"fmt"
	"os/exec"
)

func (c *CLI) openURL(url string) error {
	switch c.os {
	case Linux:
		cmd := exec.Command("xdg-open", url)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to open domain %s", url)
		}
	case MacOS:
		cmd := exec.Command("open", url)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to open domain %s", url)
		}
	case Windows:
		cmd := exec.Command("start", url)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Failed to open domain %s", url)
		}
	}

	return nil
}
