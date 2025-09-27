package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/caio-ishikawa/scopewarden/shared/models"
	"net/http"
	"net/url"
	"runtime"
	"strconv"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
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

var (
	// Default URL table columns
	StatsColumns = []table.Column{
		{Title: "Total Found URLs", Width: 16},
		{Title: "Total New URLs", Width: 14},
		{Title: "Total Found Ports", Width: 17},
		{Title: "Total New Ports", Width: 15},
		{Title: "Scan duration", Width: 13},
		{Title: "Scan time", Width: 20},
		{Title: "Is Running", Width: 10},
	}

	URLColumns = []table.Column{
		{Title: "Status", Width: 6},
		{Title: "Ports", Width: 5},
		{Title: "Brute", Width: 5},
		{Title: "URL", Width: 63},
	}

	PortColumns = []table.Column{
		{Title: "Ports", Width: 10},
		{Title: "Protocol", Width: 15},
		{Title: "State", Width: 19},
	}

	BruteForcedColumns = []table.Column{
		{Title: "Assets", Width: 22},
	}
)

type CLIState string
type OperatingSystem string

const (
	TargetDomainTable CLIState = "DomainTable"
	PortsTable        CLIState = "PortState"
	StatsTable        CLIState = "StatsTable"
	BruteForcedTable  CLIState = "BruteForcedTable"
	SortMode          CLIState = "Sort"

	Linux   OperatingSystem = "Linux"
	MacOS   OperatingSystem = "MacOS"
	Windows OperatingSystem = "Windows"
)

// Map of domain URL to PerDomainRow
type DomainRows map[string]PerDomainRow

type PerDomainRow struct {
	Port        []table.Row
	BruteForced []table.Row
}

type CLI struct {
	table             table.Model
	portsTable        table.Model
	bruteForcedTable  table.Model
	domainMap         DomainRows
	help              help.Model
	os                OperatingSystem
	sortBy            models.DomainSortBy
	selectedDomainURL string
	selectedDomainIdx int
	domainOffset      int
	bruteForcedOffset int
	targetUUID        string
	state             CLIState
}

func NewCLI() (CLI, error) {
	mainTable := table.New()
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

	//mainTable.SetStyles(s)
	mainTable.SetHeight(tableHeightHalfScreen)

	portsTable := table.New()
	//portsTable.SetStyles(s)
	portsTable.SetHeight(tableHeightHalfScreen)

	bruteForcedTable := table.New()
	//bruteForcedTable.SetStyles(s)
	bruteForcedTable.SetHeight(tableHeightHalfScreen)

	var operatingSystem OperatingSystem
	switch runtime.GOOS {
	case "linux":
		operatingSystem = Linux
	case "darwin":
		operatingSystem = MacOS
	case "windows":
		operatingSystem = Windows
	default:
		return CLI{}, fmt.Errorf("Unsupported OS: %s", runtime.GOOS)
	}

	return CLI{
		table:             mainTable,
		portsTable:        portsTable,
		bruteForcedTable:  bruteForcedTable,
		domainMap:         map[string]PerDomainRow{},
		state:             TargetDomainTable,
		help:              help.New(),
		sortBy:            models.SortNone,
		os:                operatingSystem,
		domainOffset:      0,
		bruteForcedOffset: 0,
		selectedDomainURL: "",
		selectedDomainIdx: 0,
	}, nil
}

func (c *CLI) Init() tea.Cmd { return nil }

// TODO: make tab switch to next table to the right
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
			if m, c, skip := c.handleKeyL(); !skip {
				return m, c
			}
		case "h":
			if m, c, skip := c.handleKeyH(); !skip {
				return m, c
			}
		case "j":
			if m, c, skip := c.handleKeyJ(); !skip {
				return m, c
			}
		case "k":
			if m, c, skip := c.handleKeyK(); !skip {
				return m, c
			}
		case "p":
			if m, c, skip := c.handleKeyP(); !skip {
				return m, c
			}
		case "a":
			if m, c, skip := c.handleKeyA(); !skip {
				return m, c
			}
		case "b":
			if m, c, skip := c.handleKeyB(); !skip {
				return m, c
			}
		case "c":
			if m, c, skip := c.handleKeyC(); !skip {
				return m, c
			}
		case "s":
			if m, c, skip := c.handleKeyS(); !skip {
				return m, c
			}
		case "q":
			if m, c, skip := c.handleKeyQ(); !skip {
				return m, c
			}
		case "tab":
			if m, c, skip := c.handleKeyTab(); !skip {
				return m, c
			}
		case "ctrl+c":
			return c, tea.Quit
		case "enter":
			if m, c, skip := c.handleKeyEnter(); !skip {
				return m, c
			}
		}
	}
	c.table, cmd = c.table.Update(msg)

	return c, cmd
}

func (c *CLI) View() string {
	return c.updateStyles()
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

	c.table.SetColumns(URLColumns)
	c.table.SetRows(rows)
	c.table.SetCursor(0)
	c.selectedDomainURL = c.table.SelectedRow()[3]

	// Get associated domain rows
	perDomainRows, ok := c.domainMap[c.selectedDomainURL]
	if !ok {
		panic(fmt.Sprintf("Could not find associated domain rows for %s", c.selectedDomainURL))
	}

	c.portsTable.SetColumns(PortColumns)
	c.portsTable.SetRows(perDomainRows.Port)

	c.bruteForcedTable.SetColumns(BruteForcedColumns)
	c.bruteForcedTable.SetRows(perDomainRows.BruteForced)

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

	c.table.SetHeight(tableHeightOne)
	c.table.SetColumns(StatsColumns)
	c.table.SetRows(rows)

	fmt.Println(c.View())
	return nil
}

// Gets domains and creates map for domain to associated rows (ports & bruteforced)
func (c *CLI) GetDomainRows() ([]table.Row, error) {
	res, err := GetDomainsByTarget(c.targetUUID, c.domainOffset, c.sortBy)
	if err != nil {
		return nil, fmt.Errorf("Failed to get domains rows: %w", err)
	}

	output := make([]table.Row, 0)
	for _, domain := range res.Domains {
		var row PerDomainRow

		domainRow := table.Row{
			strconv.Itoa(domain.StatusCode),
			strconv.Itoa(domain.PortCount),
			strconv.Itoa(domain.BruteForcedCount),
			domain.URL,
		}

		for _, port := range domain.Ports {
			row.Port = append(
				row.Port,
				table.Row{
					strconv.Itoa(port.Port),
					string(port.Protocol),
					string(port.State),
				},
			)
		}

		for _, bruteForced := range domain.BruteForced {
			row.BruteForced = append(
				row.BruteForced,
				table.Row{
					bruteForced.Path,
				},
			)
		}

		output = append(output, domainRow)
		c.domainMap[domain.URL] = row
	}

	return output, nil
}

func (c *CLI) GetPortRows(ports []models.Port) ([]table.Row, error) {
	var rows []table.Row
	for _, port := range ports {
		portStr := strconv.Itoa(port.Port)
		if portStr == "" {
			return nil, fmt.Errorf("Failed to get rows for port: Could not convert port number to string %v", port.Port)
		}

		rows = append(rows, table.Row{portStr, string(port.Protocol), string(port.State)})
	}

	return rows, nil
}

func (c *CLI) GetBruteForcedRows(assets []models.BruteForced) ([]table.Row, error) {
	var rows []table.Row
	for _, asset := range assets {
		rows = append(rows, table.Row{asset.Path})
	}

	return rows, nil
}

func GetDomainsByTarget(target string, offset int, sortBy models.DomainSortBy) (models.DomainListResponse, error) {
	url := fmt.Sprintf("%s/domains?target_uuid=%s&limit=%v&offset=%v&sort_by=%s", apiURL, target, tableLimit, offset, sortBy)
	res, err := http.Get(url)
	if err != nil {
		return models.DomainListResponse{}, fmt.Errorf("Could not get domains for target %s: %w", target, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return models.DomainListResponse{}, fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	var ret models.DomainListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return models.DomainListResponse{}, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret, nil
}

func GetPortsByDomain(domainURL string) ([]models.Port, error) {
	param := url.Values{}
	param.Add("domain_url", domainURL)
	url := fmt.Sprintf("%s/ports?%s", apiURL, param.Encode())
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get domains for domain %s: %w", domainURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	var ret models.PortListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret.Ports, nil
}

func GetBruteForcedByDomain(domainURL string, offset int) ([]models.BruteForced, error) {
	param := url.Values{}
	param.Add("domain_url", domainURL)

	url := fmt.Sprintf("%s/bruteforced?%s&limit=%v&offset=%v", apiURL, param.Encode(), tableLimit, offset)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get domains for domain %s: %w", domainURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Unexpected error code: %v", res.StatusCode)
	}

	var ret models.BruteForcedListResponse
	if err = json.NewDecoder(res.Body).Decode(&ret); err != nil {
		return nil, fmt.Errorf("Failed to decode API response: %w", err)
	}

	return ret.BruteForcedPaths, nil
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
			TargetName: scopes.TargetName,
			URL:        scopeURL,
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

func DisableTarget(target string) error {
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
