// blocks/shell_command/renderers/gauge.go
package renderers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type GaugeRenderer struct{}

func (r *GaugeRenderer) Render(data interface{}, width int, style lipgloss.Style) string {
	metrics, ok := data.(map[string]string)
	if !ok {
		return style.Render("Error: GaugeRenderer expected map[string]string data.")
	}

	var builder strings.Builder
	barLength := 25 // Length of the progress bar

	for key, valueStr := range metrics {
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			continue // Skip non-numeric values
		}

		// Clamp value between 0 and 100
		if value < 0 {
			value = 0
		}
		if value > 100 {
			value = 100
		}

		filledCount := int((value / 100.0) * float64(barLength))
		emptyCount := barLength - filledCount

		//bar := strings.Repeat("█", filledCount) + strings.Repeat("░", emptyCount)
		
		// Style the bar
		filledStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("154")) // Greenish
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))  // Gray

		styledBar := filledStyle.Render(strings.Repeat("█", filledCount)) + emptyStyle.Render(strings.Repeat("░", emptyCount))


		label := fmt.Sprintf("%-5s", strings.ToUpper(key))
		percent := fmt.Sprintf("%5.1f%%", value)

		builder.WriteString(fmt.Sprintf("%s [%s] %s\n", label, styledBar, percent))
	}

	return style.Render(builder.String())
}