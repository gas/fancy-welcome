// shared/block/block.go
package block

import (
	"time"
	"github.com/charmbracelet/bubbletea"
	//"github.com/charmbracelet/lipgloss"
	"github.com/gas/fancy-welcome/config" // Importamos el paquete de config para uso particular
	"github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/logging"
)

type TriggerUpdateMsg struct{}

// BlockTickMsg es el mensaje que se enviará periódicamente a un bloque específico.
type BlockTickMsg struct {
	TargetBlockID string
}
// Hacemos que cumpla la interfaz para ser un mensaje dirigido.
func (m BlockTickMsg) BlockID() string { return m.TargetBlockID }

// Block es la interfaz que cada módulo de bloque debe implementar.
type Block interface {
	Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error
	// El nuevo Update recibe el mensaje y devuelve el bloque actualizado y un comando.
	Update(msg tea.Msg) (Block, tea.Cmd)
	View() string
	Name() string
	// helper de posición
    Position() string
}

// ScheduleNextUpdate devuelve un comando que envía un TriggerUpdateMsg después de un intervalo.
func ScheduleNextUpdate(blockID string, interval time.Duration) tea.Cmd {
	logging.Log.Printf(">>> Scheduling next TICK for [%s] in %v", blockID, interval)
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return BlockTickMsg{TargetBlockID: blockID}
	})
}

// ScheduleNextUpdate devuelve un comando que envía un TriggerUpdateMsg después de un intervalo.
func ScheduleNextTick(blockID string, interval time.Duration) tea.Cmd {
	logging.Log.Printf(">>> Scheduling next TICK for [%s] in %v", blockID, interval)
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return BlockTickMsg{TargetBlockID: blockID}
	})
}

type TargetedMsg interface {
	BlockID() string
}