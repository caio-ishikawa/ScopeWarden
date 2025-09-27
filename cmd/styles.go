package main

import (
	//"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"

	"github.com/charmbracelet/lipgloss"
)

const (
	green = "#A7C080"
	black = "#1E2326"
	grey  = "#384B55"
	white = "#F2EFDF"
	red   = "#E67E80"

	tableLimit            = 39
	tableHeightHalfScreen = 40
	tableHeightFullScreen = 80
	tableHeightOne        = 2

	apiURL = "http://localhost:8080"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(green))

type TableBoxStyles struct {
	Active   lipgloss.Style
	Inactive lipgloss.Style
}

var (
	DefaultBoxStyles = TableBoxStyles{
		Active: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(green)).
			Margin(0, 0),
		Inactive: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(grey)).
			Margin(0, 0),
	}
)

// Updates tables styles to represent currently active table
// TODO: This needs to not render other tables if we're rendering the stats only
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
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			DefaultBoxStyles.Inactive.Render(c.table.View()),
			DefaultBoxStyles.Active.Render(c.portsTable.View()),
			DefaultBoxStyles.Inactive.Render(c.bruteForcedTable.View()),
		)

		return c.getView(row)
	case BruteForcedTable:
		c.bruteForcedTable.SetStyles(selectedStyle)
		c.table.SetStyles(inactiveStyle)
		c.portsTable.SetStyles(inactiveStyle)
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			DefaultBoxStyles.Inactive.Render(c.table.View()),
			DefaultBoxStyles.Inactive.Render(c.portsTable.View()),
			DefaultBoxStyles.Active.Render(c.bruteForcedTable.View()),
		)

		return c.getView(row)
	case StatsTable:
		c.table.SetStyles(selectedStyle)
		return DefaultBoxStyles.Active.Render(c.table.View())
	// For undefined state or TargetDomainTable
	default:
		c.table.SetStyles(selectedStyle)
		c.portsTable.SetStyles(inactiveStyle)
		c.bruteForcedTable.SetStyles(inactiveStyle)
		row := lipgloss.JoinHorizontal(
			lipgloss.Top,
			DefaultBoxStyles.Active.Render(c.table.View()),
			DefaultBoxStyles.Inactive.Render(c.portsTable.View()),
			DefaultBoxStyles.Inactive.Render(c.bruteForcedTable.View()),
		)

		return c.getView(row)
	}
}

func (c *CLI) getView(row string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		row,
		c.help.View(c),
	)
}
