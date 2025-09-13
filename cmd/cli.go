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
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(green))

type CLI struct {
	db store.Database
}

type model struct {
	table table.Model
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			// TODO: open link
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func NewCLI() (CLI, error) {
	db, err := store.Init()
	if err != nil {
		return CLI{}, fmt.Errorf("Failed to start DB client: %w", err)
	}

	return CLI{
		db: db,
	}, nil
}

func (c CLI) RenderURLsTable(targetUUID string) error {
	offset := 0
	domains, err := c.db.GetDomainsPerTarget(50, offset, targetUUID)
	if err != nil {
		return fmt.Errorf("Failed to get domains by target: %w", err)
	}

	columns := []table.Column{
		{Title: "STATUS", Width: 6},
		{Title: "URL", Width: 129},
		{Title: "QUERY PARAMS", Width: 30},
	}

	var rows []table.Row
	for _, domain := range domains {
		rows = append(rows, table.Row{strconv.Itoa(domain.StatusCode), domain.URL, domain.QueryParams})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(40),
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

	m := model{t}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return fmt.Errorf("Error rendering domain table: %w", err)
	}

	return nil
}
