// blocks/shell_command/renderers/cowsay.go
package renderers

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
    "github.com/gas/fancy-welcome/utils"
)

type CowsayRenderer struct{}

func (r *CowsayRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
    message, ok := data.(string)
    if !ok {
        return style.Render(fmt.Sprintf("Error: CowsayRenderer esperaba un string, recibió %T", data))
    }
    return style.Render(utils.Generate(message, width))
}
