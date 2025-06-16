// blocks/shell_command/renderers/raw_text.go
package renderers

import (
    "fmt"
    //"strings"
    "github.com/charmbracelet/lipgloss"
)

type RawTextRenderer struct{}

func (r *RawTextRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
    if text, ok := data.(string); ok {
        return style.Render(text)
    }
    return style.Render(fmt.Sprintf("Error: RawTextRenderer recibió datos incompatibles de tipo %T", data))
}
