// blocks/shell_command/renderers/table.go
package renderers

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderTableHelper es una función interna para no duplicar código.
func renderTable(tableData [][]string, style lipgloss.Style) string {
	if len(tableData) == 0 {
		return ""
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
	headerStyle := style.Copy().Bold(true).Foreground(lipgloss.Color("12"))

	// Render header
	header := tableData[0]
	for i, cell := range header {
		builder.WriteString(headerStyle.Width(colWidths[i]).Render(cell))
		builder.WriteString("  ")
	}
	builder.WriteString("\n")

	// Render body
	for _, row := range tableData[1:] {
		for i, cell := range row {
			builder.WriteString(style.Width(colWidths[i]).Render(cell))
			builder.WriteString("  ")
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

type TableRenderer struct{}

func (r *TableRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
	// Intenta la conversión directa primero
	if tableData, ok := data.([][]string); ok {
		return renderTable(tableData, style)
	}

	// ---- NUEVA LÓGICA PARA DATOS DE CACHÉ ----
	// Si falla, intenta convertir desde el formato de la caché JSON
	if interfaceSlice, ok := data.([]interface{}); ok {
		var tableData [][]string
		for _, rowInterface := range interfaceSlice {
			if rowSlice, ok := rowInterface.([]interface{}); ok {
				var row []string
				for _, cellInterface := range rowSlice {
					if cell, ok := cellInterface.(string); ok {
						row = append(row, cell)
					}
				}
				if len(row) > 0 {
					tableData = append(tableData, row)
				}
			}
		}
		// Una vez convertidos los datos, renderiza la tabla
		return renderTable(tableData, style)
	}
	// ---- FIN DE LA NUEVA LÓGICA ----

	return style.Render(fmt.Sprintf("Error: TableRenderer received incompatible data type %T", data))
}