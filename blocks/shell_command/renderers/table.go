// blocks/shell_command/renderers/table.go
package renderers

import (
	//"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TableRenderer struct{}

func (r *TableRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
	tableData, ok := data.([][]string)
	if !ok || len(tableData) == 0 {
		return style.Render("Error: TableRenderer expected [][]string data.")
	}

	// Calculate column widths
	numCols := len(tableData[0])
	colWidths := make([]int, numCols)
	for _, row := range tableData {
		if len(row) == numCols {
			for i, cell := range row {
				if len(cell) > colWidths[i] {
					colWidths[i] = len(cell)
				}
			}
		}
	}

	var builder strings.Builder
	
	// Header style
	headerStyle := style.Copy().Bold(true).Foreground(lipgloss.Color("12")) // A distinct color for headers

	// Render header
	header := tableData[0]
	for i, cell := range header {
		builder.WriteString(headerStyle.Width(colWidths[i]).Render(cell))
		builder.WriteString("  ") // Padding
	}
	builder.WriteString("\n")

	// Render body
	for _, row := range tableData[1:] {
		for i, cell := range row {
			builder.WriteString(style.Width(colWidths[i]).Render(cell))
			builder.WriteString("  ") // Padding
		}
		builder.WriteString("\n")
	}

	return builder.String()
}