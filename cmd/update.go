package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"net/url"
	"os/exec"
)

func (c *CLI) handleKeyL() (tea.Model, tea.Cmd, bool) {
	if c.state == TargetDomainTable {
		c.domainOffset = c.domainOffset + tableLimit
		rows, err := c.GetDomainRows()
		if err != nil {
			tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
			return c, tea.Quit, false
		}
		c.table.SetRows(rows)
	}
	if c.state == BruteForcedTable {
		c.bruteForcedOffset = c.bruteForcedOffset + tableLimit

		bruteForced, err := GetBruteForcedByDomain(c.selectedDomainURL, c.bruteForcedOffset)
		if err != nil {
			tea.Println("ERROR: COULD NOT GET BRUTE FORCED DOMAINS")
			return c, tea.Quit, false
		}

		rows, err := c.GetBruteForcedRows(bruteForced)
		if err != nil {
			tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
			return c, tea.Quit, false
		}

		c.table.SetRows(rows)

		c.selectedDomainIdx = 0
		//helpView := c.help.View("")
	}

	return nil, nil, true
}

func (c *CLI) handleKeyH() (tea.Model, tea.Cmd, bool) {
	if c.state == TargetDomainTable {
		if c.domainOffset >= tableLimit {
			c.domainOffset = c.domainOffset - tableLimit

			rows, err := c.GetDomainRows()
			if err != nil {
				tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
				return c, tea.Quit, false
			}

			c.table.SetRows(rows)

			c.selectedDomainIdx = 0
		}
	}
	if c.state == BruteForcedTable {
		if c.bruteForcedOffset >= tableLimit {
			c.bruteForcedOffset = c.bruteForcedOffset - tableLimit

			bruteForced, err := GetBruteForcedByDomain(c.selectedDomainURL, c.bruteForcedOffset)
			if err != nil {
				tea.Println("ERROR: COULD NOT GET BRUTE FORCED DOMAINS")
				return c, tea.Quit, false
			}

			rows, err := c.GetBruteForcedRows(bruteForced)
			if err != nil {
				tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
				return c, tea.Quit, false
			}

			c.table.SetRows(rows)
		}
	}

	return nil, nil, true
}

func (c *CLI) handleKeyJ() (tea.Model, tea.Cmd, bool) {
	c.table.MoveDown(1)

	if c.state == TargetDomainTable {
		c.selectedDomainURL = c.table.SelectedRow()[3]
		if c.selectedDomainIdx < tableLimit {
			c.selectedDomainIdx += 1
		}
	}

	return nil, nil, true
}

func (c *CLI) handleKeyK() (tea.Model, tea.Cmd, bool) {
	c.table.MoveUp(1)

	if c.state == TargetDomainTable {
		c.selectedDomainURL = c.table.SelectedRow()[3]
		if c.selectedDomainIdx > 0 {
			c.selectedDomainIdx -= 1
		}
	}

	return nil, nil, true
}

func (c *CLI) handleKeyP() (tea.Model, tea.Cmd, bool) {
	if c.state == TargetDomainTable {
		c.table.SetCursor(0)

		ports, err := GetPortsByDomain(c.selectedDomainURL)
		if err != nil {
			tea.Printf("Failed to get ports by domain: %s", err.Error())
			return c, tea.Quit, false
		}

		rows, err := c.GetPortRows(ports)
		if err != nil {
			tea.Printf("Failed to get rows and columns for ports: %s", err.Error())
			return c, tea.Quit, false
		}

		if len(rows) == 0 {
			return nil, nil, true
		}

		c.state = PortsTable

		// Render port table
		c.table.SetRows(rows)
		c.table.SetColumns(PortColumns)
	}

	return nil, nil, true
}

func (c *CLI) handleKeyA() (tea.Model, tea.Cmd, bool) {
	if c.state == TargetDomainTable {
		c.table.SetCursor(0)

		bruteForcedPaths, err := GetBruteForcedByDomain(c.selectedDomainURL, c.bruteForcedOffset)
		if err != nil {
			tea.Printf("Failed to get bruteforced assets by domain: %s", err.Error())
			return c, tea.Quit, false
		}

		rows, err := c.GetBruteForcedRows(bruteForcedPaths)
		if err != nil {
			tea.Printf("Failed to get rows for bruteforced assets: %s", err.Error())
			return c, tea.Quit, false
		}

		if len(rows) == 0 {
			return nil, nil, true
		}

		c.state = BruteForcedTable

		// Render bruteforced table
		c.table.SetRows(rows)
		c.table.SetColumns(BruteForcedColumns)
	}

	return nil, nil, true
}

func (c *CLI) handleKeyB() (tea.Model, tea.Cmd, bool) {
	// Go back to URL table from other tables
	if c.state == PortsTable || c.state == BruteForcedTable {
		rows, err := c.GetDomainRows()
		if err != nil {
			tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
			return c, tea.Quit, false
		}

		c.state = TargetDomainTable

		// Render the URL rows from previously recorded offset
		c.table.SetColumns(URLColumns)
		c.table.SetRows(rows)
		c.table.SetCursor(c.selectedDomainIdx)
	}

	return nil, nil, true
}

func (c *CLI) handleKeyQ() (tea.Model, tea.Cmd, bool) {
	// Go back to URL table from other tables
	if c.state == PortsTable || c.state == BruteForcedTable {
		c.bruteForcedOffset = 0

		rows, err := c.GetDomainRows()
		if err != nil {
			tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
			return c, tea.Quit, false
		}

		c.state = TargetDomainTable

		// Render the URL rows from previously recorded offset
		c.table.SetColumns(URLColumns)
		c.table.SetRows(rows)
		c.table.SetCursor(c.selectedDomainIdx)
	} else {
		return c, tea.Quit, false
	}

	return nil, nil, true
}

func (c *CLI) handleKeyEnter() (tea.Model, tea.Cmd, bool) {
	if c.state == TargetDomainTable {
		if err := c.openURL(c.selectedDomainURL); err != nil {
			tea.Println(err.Error())
		}
	}
	if c.state == BruteForcedTable {
		bruteForced := c.table.SelectedRow()[0]
		var err error

		// Validate if we can implicitly create a URL to attempt opening on the browser based on the brute forced path
		if bruteForced[0] == '/' {
			err = c.openURL(fmt.Sprintf("%s%s", c.selectedDomainURL, bruteForced))
		} else if _, err := url.Parse(bruteForced); err != nil {
			err = c.openURL(bruteForced)
		} else {
			err = c.openURL(fmt.Sprintf("%s/%s", c.selectedDomainURL, bruteForced))
		}

		if err != nil {
			tea.Println(err.Error())
		}
	}

	return nil, nil, true
}

func (c *CLI) handleKeyC() (tea.Model, tea.Cmd, bool) {
	// Copy domain URL to clipboard
	if c.state == TargetDomainTable {
		switch c.os {
		case Linux:
			// Running this with 'echo' instead would copy an extra newline at the end
			cmd := exec.Command("printf", "%s", c.selectedDomainURL, "|", "xclip", "-selection", "clipboard")
			if err := cmd.Run(); err != nil {
				tea.Printf("Failed to open domain %s", c.selectedDomainURL)
			}
		case MacOS:
			cmd := exec.Command("echo", c.selectedDomainURL, "|", "pbcopy")
			if err := cmd.Run(); err != nil {
				tea.Printf("Failed to open domain %s", c.selectedDomainURL)
			}
		case Windows:
			cmd := exec.Command(c.selectedDomainURL, "|", "clip")
			if err := cmd.Run(); err != nil {
				tea.Printf("Failed to open domain %s", c.selectedDomainURL)
			}
		}
	}
	return nil, nil, true
}
