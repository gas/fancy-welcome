// blocks/shell_command/renderers/renderer.go
package renderers

import "github.com/charmbracelet/lipgloss"

// Renderer es la interfaz que cada módulo de visualización debe implementar.
// Toma datos estructurados y los convierte en un string para la TUI.
type Renderer interface {
    // Render toma los datos parseados y devuelve el string final formateado.
    Render(data interface{}, width int, style lipgloss.Style) string
}
