// shared/block/block.go
package block

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Block es la interfaz que cada módulo de bloque debe implementar.
type Block interface {
	// Name devuelve el nombre único del bloque (ej. "system_info").
	Name() string

	// Init se llama una vez al inicio para pasar la configuración específica
	// del bloque y el tema actual.
	Init(config map[string]interface{}, theme lipgloss.Style) error

	// Update se llama para refrescar los datos del bloque.
	// Debe ser una operación no bloqueante y devolver un tea.Cmd si es necesario.
	Update() tea.Cmd

	// View genera la cadena de texto a renderizar para el bloque.
	View() string
}
