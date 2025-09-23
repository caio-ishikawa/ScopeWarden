package main

import (
	tea "github.com/charmbracelet/bubbletea"
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
	// TODO: return early if no ports
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
	// TODO: return early if no bruteforced
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
