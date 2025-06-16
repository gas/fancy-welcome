// shared/block/block.go
package block

import (
	"github.com/charmbracelet/bubbletea"
	//"github.com/charmbracelet/lipgloss"
	"github.com/gas/fancy-welcome/config" // Importamos el paquete de config para uso particular
	"github.com/gas/fancy-welcome/themes"
)

// Block es la interfaz que cada módulo de bloque debe implementar.
type Block interface {
	// Name devuelve el nombre único del bloque (ej. "system_info").
	Name() string

	// Init se llama una vez al inicio para pasar la configuración específica
	// del bloque y el tema actual.
	Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error
	// Update se llama para refrescar los datos del bloque.
	// Debe ser una operación no bloqueante y devolver un tea.Cmd si es necesario.
	Update() tea.Cmd

	// View genera la cadena de texto a renderizar para el bloque.
	View() string
	RenderedHeight() int      // Devuelve la altura de la última vista renderizada
	GetPosition() string      // Nuevo método para obtener la posición
	SetWidth(width int)       // Crucial para el layout
    GetSetWidth() int         // Nuevo método para obtener el ancho que se le asignó
    GetThemeColors() themes.ThemeColors // Nuevo método para obtener los colores del tema inyectados
   	// Nuevo método: informa al modelo si el bloque ha cambiado de tal forma que
    // el dashboard necesita re-renderizarse (ej. cambió su contenido o altura).
    HasContentChanged() bool
    ResetContentChangedFlag() // Resetea el flag después de que el dashboard lo procesa
}

