package main

import (
	"github.com/charmbracelet/bubbles/table"
	//tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TableBoxStyles struct {
	Active   lipgloss.Style
	Inactive lipgloss.Style
}

var (
	DefaultBoxStyles = TableBoxStyles{
		Active: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(green)).
			Padding(0, 1),
		Inactive: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(grey)).
			Padding(0, 1),
	}
)

// Updates tables styles to represent currently active table
func (c *CLI) updateStyles() string {
	// TODO: There has to be a better way to do this, but I can't think of it right now
	selectedStyle := table.DefaultStyles()
	selectedStyle.Header = selectedStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(green)).
		BorderBottom(true).
		Bold(false)
	selectedStyle.Selected = selectedStyle.Selected.
		Foreground(lipgloss.Color(black)).
		Background(lipgloss.Color(green)).
		Bold(true)

	inactiveStyle := table.DefaultStyles()
	inactiveStyle.Header = inactiveStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(grey)).
		BorderBottom(true).
		Bold(false)
	inactiveStyle.Selected = inactiveStyle.Selected.
		Foreground(lipgloss.Color(white)).
		Background(lipgloss.Color(grey)).
		Bold(true)

	switch c.state {
	case PortsTable:
		c.portsTable.SetStyles(selectedStyle)
		c.table.SetStyles(inactiveStyle)
		c.bruteForcedTable.SetStyles(inactiveStyle)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			DefaultBoxStyles.Inactive.Render(c.table.View()),
			DefaultBoxStyles.Active.Render(c.portsTable.View()),
			DefaultBoxStyles.Inactive.Render(c.bruteForcedTable.View()),
		)
	case BruteForcedTable:
		c.bruteForcedTable.SetStyles(selectedStyle)
		c.table.SetStyles(inactiveStyle)
		c.portsTable.SetStyles(inactiveStyle)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			DefaultBoxStyles.Inactive.Render(c.table.View()),
			DefaultBoxStyles.Inactive.Render(c.portsTable.View()),
			DefaultBoxStyles.Active.Render(c.bruteForcedTable.View()),
		)
	// For undefined state or TargetDomainTable
	default:
		c.table.SetStyles(selectedStyle)
		c.portsTable.SetStyles(inactiveStyle)
		c.bruteForcedTable.SetStyles(inactiveStyle)
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			DefaultBoxStyles.Active.Render(c.table.View()),
			DefaultBoxStyles.Inactive.Render(c.portsTable.View()),
			DefaultBoxStyles.Inactive.Render(c.bruteForcedTable.View()),
		)
	}
}
