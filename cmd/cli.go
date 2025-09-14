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
	tableHeightOne        = 2

	apiURL = "http://localhost:8080"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color(green))

type CLIState string

const (
	TargetDomainTable CLIState = "DomainTable"
	StatsTable        CLIState = "StatsTable"
)

type CLI struct {
	table      table.Model
	offset     int
	targetUUID string
	state      CLIState
}

func (c *CLI) Init() tea.Cmd { return nil }

func (c *CLI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if c.state == TargetDomainTable {
				c.offset = c.offset + tableLimit
				rows, err := c.GetDomainRows()
				if err != nil {
					tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
					return c, tea.Quit
				}
				c.table.SetRows(rows)
			}
		case "h":
			if c.state == TargetDomainTable {
				if (c.offset - tableLimit) >= tableLimit {
					c.offset = c.offset - tableLimit
					tea.Println(c.offset)
					rows, err := c.GetDomainRows()
					if err != nil {
						tea.Println("ERROR: COULD NOT UPDATE DOMAINS")
						return c, tea.Quit
					}
					c.table.SetRows(rows)
				}
			}
		case "j":
			if c.state == TargetDomainTable {
				c.table.MoveDown(1)
			}
		case "k":
			if c.state == TargetDomainTable {
				c.table.MoveUp(1)
			}
		case "q", "ctrl+c":
			return c, tea.Quit
		case "enter":
			if c.state == TargetDomainTable {
				// TODO: open link
				return c, tea.Batch(
					tea.Printf("Let's go to %s!", c.table.SelectedRow()[1]),
				)
			}
		}
	}
	c.table, cmd = c.table.Update(msg)

	return c, cmd
}

func (m *CLI) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func NewCLI() (CLI, error) {
	t := table.New()
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
		table:  t,
		offset: 0,
	}, nil
}

func (c *CLI) SetTarget(targetName string) error {
	target, err := GetTargetByName(targetName)
	if err != nil {
		return fmt.Errorf("Failed to get target by name: %w", err)
	}

	c.targetUUID = target.UUID

	return nil
}

func (c *CLI) RenderURLsTable() error {
	c.state = TargetDomainTable

	rows, err := c.GetDomainRows()
	if err != nil {
		return fmt.Errorf("Error getting domain rows: %w", err)
	}

	columns := []table.Column{
		{Title: "STATUS", Width: 6},
		{Title: "URL", Width: 129},
		{Title: "QUERY PARAMS", Width: 30},
	}

	c.table.SetHeight(tableHeightHalfScreen)
	c.table.SetColumns(columns)
	c.table.SetRows(rows)

	if _, err := tea.NewProgram(c).Run(); err != nil {
		return fmt.Errorf("Error rendering stats table: %w", err)
	}

	return nil
}

func (c *CLI) RenderStatsTable() error {
	c.state = StatsTable

	stats, err := GetStats()
	if err != nil {
		return fmt.Errorf("Failed to get stats: %w", err)
	}

	rows := []table.Row{
		{
			strconv.Itoa(stats.TotalFoundURLs),
			strconv.Itoa(stats.TotalNewURLs),
			strconv.Itoa(stats.TotalFoundPorts),
			strconv.Itoa(stats.TotalNewPorts),
			stats.ScanTime,
			stats.ScanBegin,
			strconv.FormatBool(stats.IsRunning),
		},
	}

	columns := []table.Column{
		{Title: "Total Found URLs", Width: 16},
		{Title: "Total New URLs", Width: 14},
		{Title: "Total Found Ports", Width: 17},
		{Title: "Total New Ports", Width: 15},
		{Title: "Scan duration", Width: 13},
		{Title: "Scan time", Width: 20},
		{Title: "Is Running", Width: 10},
	}

	c.table.SetHeight(tableHeightOne)
	c.table.SetColumns(columns)
	c.table.SetRows(rows)

	if _, err := tea.NewProgram(c).Run(); err != nil {
		return fmt.Errorf("Error rendering stats table: %w", err)
	}

	return nil
}

func (c *CLI) GetDomainRows() ([]table.Row, error) {
	domains, err := GetDomainsByTarget(c.targetUUID, c.offset)
	if err != nil {
		return nil, fmt.Errorf("Failed to get domains rows: %w", err)
	}

	var rows []table.Row
	for _, domain := range domains {
		rows = append(rows, table.Row{strconv.Itoa(domain.StatusCode), domain.URL, domain.QueryParams})
	}

	return rows, nil
}

func GetDomainsByTarget(target string, offset int) ([]models.Domain, error) {
	url := fmt.Sprintf("%s/domains?target_uuid=%s&limit=%v&offset=%v", apiURL, target, tableLimit, offset)
	res, err := http.Get(url)
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

func GetStats() (models.StatsResponse, error) {
	res, err := http.Get(fmt.Sprintf("%s/stats", apiURL))
	if err != nil {
		return models.StatsResponse{}, fmt.Errorf("Could not get stats: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return models.StatsResponse{}, fmt.Errorf("Unexpected status code: %v", res.StatusCode)
	}

	var ret models.StatsResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return models.StatsResponse{}, fmt.Errorf("Failed to decode API response: %w", err)
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
