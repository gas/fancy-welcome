// shared/layout.go
package shared

import (
	"github.com/charmbracelet/lipgloss"
    "github.com/gas/fancy-welcome/logging"
	"github.com/gas/fancy-welcome/shared/block"
)

// RenderDashboard compone la vista de todos los bloques en un solo string.
// Recibe todo lo que necesita para renderizar como argumentos.
func RenderDashboard(
	width int,
	blocks []block.Block,
	focusIndex int,
	normalStyle, focusStyle lipgloss.Style,
) string {
	if width == 0 {
		return "Initializing..."
	}

	var finalLayout []string
	var leftColumnViews []string
	var rightColumnViews []string

	processPendingColumns := func() {
        if len(leftColumnViews) > 0 || len(rightColumnViews) > 0 {
            leftColumn := lipgloss.JoinVertical(lipgloss.Left, leftColumnViews...)
            rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightColumnViews...)
            
            // Une las columnas horizontalmente
            joinedCols := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
            finalLayout = append(finalLayout, joinedCols)

            // Limpia los slices para la siguiente sección de columnas
            leftColumnViews = []string{}
            rightColumnViews = []string{}
        }
	}

	for i, b := range blocks {
		blockView := b.View()
		var renderedBlock string

        // --- LÓGICA DE RENDERIZADO CONDICIONAL PARA TEXTO CRUDO---
        // Ya que hay casos de texto que vienen con color (neofetch, etc) y no queremos
        // que se le aplique el style ya que lo pone blanco.
        // Tiene que haber una mejor forma de hacerlo... porque así ni le pone el borde
        if b.RendererName() == "preformatted_text" {
            // Si es pre-formateado, usamos la vista cruda, sin bordes ni estilos.
            renderedBlock = blockView
            logging.Log.Printf("PREFORMATTED: blockIndex=%d, blockName=%s", i, b.Name())

        } else {

			var borderStyle lipgloss.Style

			if i == focusIndex {
				borderStyle = focusStyle
			} else {
				borderStyle = normalStyle
			}

			position := b.Position()
			if position == "left" || position == "right" {
				blockWidth := (width / 2) - 4
				renderedBlock = borderStyle.Width(blockWidth).Render(blockView)
			} else {
				blockWidth := width - 2
				renderedBlock = borderStyle.Width(blockWidth).Render(blockView)
			}
		}

	    position := b.Position()
	    if position == "left" || position == "right" {
	         if position == "left" {
	            leftColumnViews = append(leftColumnViews, renderedBlock)
	        } else {
	            rightColumnViews = append(rightColumnViews, renderedBlock)
	        }
	    } else {
	         processPendingColumns()
	         finalLayout = append(finalLayout, renderedBlock)
	    }
	}

	processPendingColumns()
	return lipgloss.JoinVertical(lipgloss.Left, finalLayout...)
}