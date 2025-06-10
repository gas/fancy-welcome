// blocks/shell_command/renderers/raw_text.go
package renderers

import (
    //"fmt"
    "github.com/charmbracelet/lipgloss"
)

type RawTextRenderer struct{}

func (r *RawTextRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
    // Aseguramos que los datos son del tipo esperado (string)
    message, ok := data.(string)
    if !ok {
        return style.Render("Error: RawTextRenderer esperaba un string.")
    }
    return style.Render(message)
}
