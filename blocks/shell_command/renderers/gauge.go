// blocks/shell_command/renderers/gauge.go
package renderers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderGaugeHelper es una función interna para no duplicar código.
func renderGauge(metrics map[string]string, style lipgloss.Style) string {
	var builder strings.Builder
	barLength := 25

	for key, valueStr := range metrics {
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue
		}

		if value < 0 { value = 0 }
		if value > 100 { value = 100 }

		filledCount := int((value / 100.0) * float64(barLength))
		
		filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("154"))
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		styledBar := filledStyle.Render(strings.Repeat("█", filledCount)) + emptyStyle.Render(strings.Repeat("░", barLength - filledCount))

		label := fmt.Sprintf("%-5s", strings.ToUpper(key))
		percent := fmt.Sprintf("%5.1f%%", value)

		builder.WriteString(fmt.Sprintf("%s [%s] %s\n", label, styledBar, percent))
	}

	return style.Render(builder.String())
}

type GaugeRenderer struct{}

func (r *GaugeRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
	// Intenta la conversión directa primero
	if metrics, ok := data.(map[string]string); ok {
		return renderGauge(metrics, style)
	}

	// ---- NUEVA LÓGICA PARA DATOS DE CACHÉ ----
	// Si falla, intenta convertir desde el formato de la caché JSON
	if interfaceMap, ok := data.(map[string]interface{}); ok {
		metrics := make(map[string]string)
		for key, valueInterface := range interfaceMap {
			if value, ok := valueInterface.(string); ok {
				metrics[key] = value
			}
		}
		// Una vez convertidos los datos, renderiza el gauge
		return renderGauge(metrics, style)
	}
	// ---- FIN DE LA NUEVA LÓGICA ----

	return style.Render(fmt.Sprintf("Error: GaugeRenderer received incompatible data type %T", data))
}