// blocks/shell_command/renderers/list.go
package renderers

import (
	"fmt"
	"strings"
	"github.com/charmbracelet/lipgloss"
)

type ListRenderer struct{}

func (r *ListRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
	var builder strings.Builder

	// Primero, intenta la conversión directa a []string
	if lines, ok := data.([]string); ok {
		for _, line := range lines {
			builder.WriteString(fmt.Sprintf("- %s\n", line))
		}
		return style.Render(builder.String())
	}
	
	// Si falla, intenta la conversión a []interface{} (típico de la caché JSON)
	if lines, ok := data.([]interface{}); ok {
		for _, lineInterface := range lines {
			// Convierte cada elemento de la interfaz a string
			if line, ok := lineInterface.(string); ok {
				builder.WriteString(fmt.Sprintf("- %s\n", line))
			}
		}
		return style.Render(builder.String())
	}

	// Si ninguna conversión funciona, muestra un error
	return style.Render(fmt.Sprintf("Error: ListRenderer received incompatible data type %T", data))
}