package main

import (
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/store"
	"strconv"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	green = "#A7C080"
	black = "#1E2326"
	red   = "#E67E80"

	tableLimit            = 50
	tableHeightHalfScreen = 40
	tableHeightFullScreen = 80
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(green))

type CLI struct {
	table      table.Model
	db         store.Database
	offset     int
	targetUUID string
}

func (c CLI) Init() tea.Cmd { return nil }

func (c CLI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if c.table.Focused() {
				c.table.Blur()
			} else {
				c.table.Focus()
			}
		case "l":
			c.offset = c.offset + tableLimit
			rows, err := c.GetDomainRows()
			if err != nil {
				tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
				return c, tea.Quit
			}
			c.table.SetRows(rows)
		case "h":
			if (c.offset - tableLimit) <= tableLimit {
				c.offset = c.offset - tableLimit
				c.RenderURLsTable()
			}
		case "q", "ctrl+c":
			return c, tea.Quit
		case "enter":
			// TODO: open link
			return c, tea.Batch(
				tea.Printf("Let's go to %s!", c.table.SelectedRow()[1]),
			)
		}
	}
	c.table, cmd = c.table.Update(msg)

	return c, cmd
}

func (m CLI) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func NewCLI(targetName string) (CLI, error) {
	db, err := store.Init()
	if err != nil {
		return CLI{}, fmt.Errorf("Failed to start DB client: %w", err)
	}

	target, err := db.GetTargetByName(targetName)
	if err != nil {
		return CLI{}, fmt.Errorf("Failed to get domain by name: %w", err)
	}

	columns := []table.Column{
		{Title: "STATUS", Width: 6},
		{Title: "URL", Width: 129},
		{Title: "QUERY PARAMS", Width: 30},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(tableHeightHalfScreen),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(green)).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(black)).
		Background(lipgloss.Color(green)).
		Bold(true)

	t.SetStyles(s)

	return CLI{
		table:      t,
		db:         db,
		targetUUID: target.UUID,
		offset:     0,
	}, nil
}

func (c CLI) GetDomainRows() ([]table.Row, error) {
	domains, err := c.db.GetDomainsPerTarget(tableLimit, c.offset, c.targetUUID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get domains by target: %w", err)
	}

	var rows []table.Row
	for _, domain := range domains {
		rows = append(rows, table.Row{strconv.Itoa(domain.StatusCode), domain.URL, domain.QueryParams})
	}

	return rows, nil
}

func (c CLI) RenderURLsTable() error {
	rows, err := c.GetDomainRows()
	if err != nil {
		return fmt.Errorf("Error getting domain rows: %w", err)
	}

	c.table.SetRows(rows)

	if _, err := tea.NewProgram(c).Run(); err != nil {
		return fmt.Errorf("Error rendering domain table: %w", err)
	}

	return nil
}
