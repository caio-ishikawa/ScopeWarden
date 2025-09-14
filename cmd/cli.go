package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/models"
	"net/http"
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

	apiURL = "http://localhost:8080"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(green))

type CLI struct {
	table      table.Model
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
	target, err := GetTargetByName(targetName)
	if err != nil {
		return CLI{}, fmt.Errorf("Failed to get target by name: %w", err)
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
		targetUUID: target.UUID,
		offset:     0,
	}, nil
}

func (c CLI) GetDomainRows() ([]table.Row, error) {
	domains, err := GetDomainsByTarget(c.targetUUID, c.offset)
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

func GetDomainsByTarget(target string, offset int) ([]models.Domain, error) {
	res, err := http.Get(fmt.Sprintf("%s/domains?target_uuid=%s&limit=%v&offset=%v", apiURL, target, tableLimit, offset))
	if err != nil {
		return nil, fmt.Errorf("Could not get domains for target %s: %w", target, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	var ret models.DomainListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret.Domains, nil
}

func GetTargetByName(target string) (models.Target, error) {
	res, err := http.Get(fmt.Sprintf("%s/target?name=%s", apiURL, target))
	if err != nil {
		return models.Target{}, fmt.Errorf("Could not get domains for target %s: %w", target, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return models.Target{}, fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	var ret models.Target
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return models.Target{}, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret, nil
}

func GetStats() (models.DaemonStats, error) {
	res, err := http.Get(fmt.Sprintf("%s/stats", apiURL))
	if err != nil {
		return models.DaemonStats{}, fmt.Errorf("Could not get stats: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return models.DaemonStats{}, fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	var ret models.DaemonStats
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return models.DaemonStats{}, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret, nil
}

func InsertScope(scopes ScopeInsert) error {
	for _, scopeURL := range scopes.ScopeURLs {
		reqBody := models.InsertScopeRequest{
			TargetName:       scopes.TargetName,
			URL:              scopeURL,
			AcceptSubdomains: scopes.AcceptSubdomains,
		}

		body, err := json.Marshal(&reqBody)
		if err != nil {
			return fmt.Errorf("Failed to marshal scope request body: %w", err)
		}

		res, err := http.Post(fmt.Sprintf("%s/insert_scope", apiURL), "application/json", bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("Could not get stats: %w", err)
		}

		if res.StatusCode != http.StatusCreated {
			return fmt.Errorf("Unexpected error code: %v", res.StatusCode)
		}
	}

	return nil
}

func InsertTarget(target string) error {
	reqBody := models.InsertTargetRequest{
		Name: target,
	}

	body, err := json.Marshal(&reqBody)
	if err != nil {
		return fmt.Errorf("Failed to marshal scope request body: %w", err)
	}

	res, err := http.Post(fmt.Sprintf("%s/insert_target", apiURL), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("Could not insert target: %w", err)
	}

	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	return nil
}
