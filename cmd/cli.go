package main

import (
	"fmt"
	"github.com/caio-ishikawa/target-tracker/shared/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"log"
)

var (
	purple    = lipgloss.Color("99")
	gray      = lipgloss.Color("245")
	lightGray = lipgloss.Color("241")

	headerStyle  = lipgloss.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
	cellStyle    = lipgloss.NewStyle().Padding(0, 1).Width(14)
	oddRowStyle  = cellStyle.Foreground(gray)
	evenRowStyle = cellStyle.Foreground(lightGray)
)

type CLI struct {
	db store.Database
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
	log.Println("man what the dfuck")
	offset := 0
	domains, err := c.db.GetDomainsPerTarget(50, offset, targetUUID)
	if err != nil {
		return fmt.Errorf("Failed to get domains by target: %w", err)
	}

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(purple)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case row%2 == 0:
				return evenRowStyle
			default:
				return oddRowStyle
			}
		}).
		Headers("URL", "QUERY PARAMS", "STATUS CODE", "LAST UPDATED").
		Rows(domains...)

	fmt.Println(t)

	return nil
}
