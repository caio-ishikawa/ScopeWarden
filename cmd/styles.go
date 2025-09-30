package main

import (
	//"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

const (
	green = "#A7C080"
	black = "#1E2326"
	grey  = "#384B55"
	grey2 = "#9DA9A0"
	white = "#D3C6AA"
	red   = "#E67E80"

	tableHeightHalfScreen = 40
	tableHeightFullScreen = 80
	tableHeightOne        = 2

	largestTableSize = 173

	apiURL = "http://localhost:8080"
)

var (
	keyMaps = map[CLIState][]key.Binding{
		TargetDomainTable: {
			key.NewBinding(key.WithKeys("j"), key.WithHelp("Move up", "k")),
			key.NewBinding(key.WithKeys("k"), key.WithHelp("Move down", "j")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("Go to Ports", "p")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("Go to Assets", "a")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("Sort by", "s")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Open URL", "Enter")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("Exit", "q/ctrl+c")),
		},
		SearchResultsTable: {
			key.NewBinding(key.WithKeys("j"), key.WithHelp("Move up", "k")),
			key.NewBinding(key.WithKeys("k"), key.WithHelp("Move down", "j")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("Go to Ports", "p")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("Go to Assets", "a")),
			key.NewBinding(key.WithKeys("s"), key.WithHelp("Sort by", "s")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Go to URL", "Enter")),
			key.NewBinding(key.WithKeys("q"), key.WithHelp("Exit search results", "q")),
		},
		PortsTable: {
			key.NewBinding(key.WithKeys("j"), key.WithHelp("Move up", "k")),
			key.NewBinding(key.WithKeys("k"), key.WithHelp("Move down", "j")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("Go to Assets", "a")),
			key.NewBinding(key.WithKeys("b"), key.WithHelp("Go back to domains table", "b/q")),
		},
		BruteForcedTable: {
			key.NewBinding(key.WithKeys("j"), key.WithHelp("Move up", "k")),
			key.NewBinding(key.WithKeys("k"), key.WithHelp("Move down", "j")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("Go to Ports", "p")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Open URL", "Enter")),
			key.NewBinding(key.WithKeys("b"), key.WithHelp("Go back to Domains", "b/q")),
		},
		SortMode: {
			key.NewBinding(key.WithKeys("p"), key.WithHelp("Sort by Ports", "p")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("Sort by Assets", "a")),
			key.NewBinding(key.WithKeys("b"), key.WithHelp("Go back to Domains", "b")),
		},
		SearchMode: {
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("Exit search", "esc")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("Search", "enter")),
		},
	}

	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(green))

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

type TableBoxStyles struct {
	Active   lipgloss.Style
	Inactive lipgloss.Style
}

// Updates tables styles to represent currently active table
// TODO: only join vertically if width larger than 256
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

	var row string
	switch c.state {
	case PortsTable:
		c.portsTable.SetStyles(selectedStyle)
		c.table.SetStyles(inactiveStyle)
		c.bruteForcedTable.SetStyles(inactiveStyle)

		if c.width >= largestTableSize {
			row = lipgloss.JoinHorizontal(
				lipgloss.Top,
				DefaultBoxStyles.Inactive.Render(c.table.View()),
				DefaultBoxStyles.Active.Render(c.portsTable.View()),
				DefaultBoxStyles.Inactive.Render(c.bruteForcedTable.View()),
			)
			break
		}

		row = DefaultBoxStyles.Active.Render(c.portsTable.View())
	case BruteForcedTable:
		c.bruteForcedTable.SetStyles(selectedStyle)
		c.table.SetStyles(inactiveStyle)
		c.portsTable.SetStyles(inactiveStyle)

		if c.width >= largestTableSize {
			row = lipgloss.JoinHorizontal(
				lipgloss.Top,
				DefaultBoxStyles.Inactive.Render(c.table.View()),
				DefaultBoxStyles.Inactive.Render(c.portsTable.View()),
				DefaultBoxStyles.Active.Render(c.bruteForcedTable.View()),
			)
			break
		}

		row = DefaultBoxStyles.Active.Render(c.bruteForcedTable.View())
	case StatsTable:
		c.table.SetStyles(selectedStyle)
		return DefaultBoxStyles.Active.Render(c.table.View())
	case SearchMode:
		searchBox := DefaultBoxStyles.Active.Render(c.searchBox.View())
		return searchBox
	// For TargetDomainTable or SearchResultsTable
	default:
		c.table.SetStyles(selectedStyle)
		c.portsTable.SetStyles(inactiveStyle)
		c.bruteForcedTable.SetStyles(inactiveStyle)

		if c.width >= largestTableSize {
			row = lipgloss.JoinHorizontal(
				lipgloss.Top,
				DefaultBoxStyles.Active.Render(c.table.View()),
				DefaultBoxStyles.Inactive.Render(c.portsTable.View()),
				DefaultBoxStyles.Inactive.Render(c.bruteForcedTable.View()),
			)
			break
		}

		row = DefaultBoxStyles.Active.Render(c.table.View())
	}

	return c.getView(row)
}

func (c *CLI) getView(row string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		row,
		c.help.View(c),
	)
}
