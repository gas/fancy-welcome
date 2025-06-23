// effects/effects.go
package effects

import "github.com/charmbracelet/bubbletea"

// Effect define el contrato para cualquier efecto visual reutilizable.
type Effect interface {
    Init() tea.Cmd
    Update(tea.Msg) (Effect, tea.Cmd)
    View() string
    IsDone() bool
    SetContent(content string)
}