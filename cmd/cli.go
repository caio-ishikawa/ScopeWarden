package main

import (
	"fmt"
	"runtime"
	"strconv"

	"github.com/caio-ishikawa/scopewarden/shared/models"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
		{Title: "URL", Width: 65},
	}

	PortColumns = []table.Column{
		{Title: "Ports", Width: 10},
		{Title: "Protocol", Width: 15},
		{Title: "State", Width: 20},
	}

	BruteForcedColumns = []table.Column{
		{Title: "Assets", Width: 25},
	}
)

type CLIState string
type OperatingSystem string

const (
	TargetDomainTable  CLIState = "DomainTable"
	PortsTable         CLIState = "PortState"
	StatsTable         CLIState = "StatsTable"
	BruteForcedTable   CLIState = "BruteForcedTable"
	SortMode           CLIState = "Sort"
	SearchMode         CLIState = "Search"
	SearchResultsTable CLIState = "SearchResultsTable"

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
	searchBox         textinput.Model
	domainMap         DomainRows
	help              help.Model
	os                OperatingSystem
	sortBy            models.DomainSortBy
	selectedDomainURL string
	selectedDomainIdx int
	domainOffset      int
	//searchResultOffset int
	bruteForcedOffset int
	targetUUID        string
	state             CLIState
	isSearching       bool
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

	helpFooter := help.New()
	helpFooter.Styles = help.Styles{
		ShortKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(grey2)),
		ShortDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(green)),

		FullKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(grey2)),
		FullDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(green)),
	}

	input := textinput.New()
	input.Placeholder = "Search (press / to focus)"
	input.CharLimit = 128
	input.Width = 40

	return CLI{
		table:             mainTable,
		portsTable:        portsTable,
		bruteForcedTable:  bruteForcedTable,
		domainMap:         map[string]PerDomainRow{},
		searchBox:         input,
		state:             TargetDomainTable,
		isSearching:       false,
		help:              helpFooter,
		sortBy:            models.SortNone,
		os:                operatingSystem,
		domainOffset:      0,
		bruteForcedOffset: 0,
		selectedDomainURL: "",
		selectedDomainIdx: 0,
	}, nil
}

func (c *CLI) Init() tea.Cmd { return nil }

func (c *CLI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if c.state == SearchMode {
		input, cmd := c.searchBox.Update(msg)
		c.searchBox = input

		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				if m, c, skip := c.handleKeyEsc(); !skip {
					return m, c
				}
			case "enter":
				if m, c, skip := c.handleKeyEnter(); !skip {
					return m, c
				}
			}

		}

		return c, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
		case "/":
			if m, c, skip := c.handleKeySlash(); !skip {
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

func (c *CLI) ShortHelp() []key.Binding {
	return keyMaps[c.state]
}

func (c *CLI) FullHelp() [][]key.Binding {
	return [][]key.Binding{keyMaps[c.state]}
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

	rows, err := c.GetDomainRows(nil)
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

	var rows []table.Row
	for _, stat := range stats {
		rows = append(rows, table.Row{
			strconv.Itoa(stat.TotalFoundURLs),
			strconv.Itoa(stat.TotalNewURLs),
			strconv.Itoa(stat.TotalFoundPorts),
			strconv.Itoa(stat.TotalNewPorts),
			stat.ScanTime,
			stat.ScanBegin,
			strconv.FormatBool(stat.IsRunning),
		})
	}

	c.table.SetHeight(tableHeightOne)
	c.table.SetColumns(StatsColumns)
	c.table.SetRows(rows)

	fmt.Println(c.View())
	return nil
}

func (c *CLI) RenderSearchModal() error {
	c.state = StatsTable

	stats, err := GetStats()
	if err != nil {
		return fmt.Errorf("Failed to get stats: %w", err)
	}

	var rows []table.Row
	for _, stat := range stats {
		rows = append(rows, table.Row{
			strconv.Itoa(stat.TotalFoundURLs),
			strconv.Itoa(stat.TotalNewURLs),
			strconv.Itoa(stat.TotalFoundPorts),
			strconv.Itoa(stat.TotalNewPorts),
			stat.ScanTime,
			stat.ScanBegin,
			strconv.FormatBool(stat.IsRunning),
		})
	}

	c.table.SetHeight(tableHeightOne)
	c.table.SetColumns(StatsColumns)
	c.table.SetRows(rows)

	fmt.Println(c.View())
	return nil
}
