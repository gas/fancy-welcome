// blocks/shell_command/renderers/preformatted_text.go
package renderers

import (
	"fmt"
	//"strings" // Asegúrate de que strings esté importado
	"github.com/charmbracelet/lipgloss"
)

type PreformattedTextRenderer struct{}

func (r *PreformattedTextRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
	if text, ok := data.(string); ok {
		return text 
	}
	// Si recibimos un tipo de dato incompatible, podemos aplicar un estilo de error.
	// Aquí sí podemos usar el estilo para el mensaje de error, ya que no es la salida del comando.
	return style.Render(fmt.Sprintf("Error: PreformattedTextRenderer received incompatible data type %T", data))
}