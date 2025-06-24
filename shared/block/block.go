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

type TeeOutputMsg struct {
	SourceBlockID string
	Output      interface{}
}

// STREAM
// StreamLineBatchMsg transporta un lote de líneas de un comando en modo streaming.
type StreamLineBatchMsg struct {
	blockID 	string
	Lines   	[]string
}
// Hacemos que cumpla la interfaz para ser un mensaje dirigido.
func (m StreamLineBatchMsg) BlockID() string { return m.blockID }

// Streamer es una interfaz que pueden implementar los bloques que necesitan
// una referencia al programa para enviar mensajes de forma continua (streaming).
type Streamer interface {
	SetProgram(p *tea.Program)
}

// BlockTickMsg es el mensaje que se enviará periódicamente a un bloque específico.
type BlockTickMsg struct {
	targetBlockID string
}
// Hacemos que cumpla la interfaz para ser un mensaje dirigido.
func (m BlockTickMsg) BlockID() string { return m.targetBlockID }

// Block es la interfaz que cada módulo de bloque debe implementar.
type Block interface {
	Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error
	// El nuevo Update recibe el mensaje y devuelve el bloque actualizado y un comando.
	Update(msg tea.Msg) (Block, tea.Cmd) //<--STREAM sin p
	View() string
	Name() string
	// helper de posición
    Position() string
	RendererName() string
}

// ScheduleNextTick devuelve un comando que envía un TriggerUpdateMsg después de un intervalo.
func ScheduleNextTick(blockID string, interval time.Duration) tea.Cmd {
	logging.Log.Printf(">>> Scheduling next TICK for [%s] in %v", blockID, interval)
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return BlockTickMsg{targetBlockID: blockID}
	})
}

type TargetedMsg interface {
	BlockID() string
}

type Expander interface {
	ExpandedView() string
}

// NewStreamLineBatchMsg es un constructor público para crear el mensaje.
func NewStreamLineBatchMsg(id string, lines []string) tea.Msg {
	return StreamLineBatchMsg{
		blockID: id,    // Correcto: esta función está DENTRO del paquete 'block'
		Lines:   lines, // Correcto: este campo es público de todas formas
	}
}

// streamClosedMsg notifica que un comando en modo streaming ha terminado o ha fallado.
type streamClosedMsg struct {
	blockID string // <--- 'b' minúscula
	err     error
}
func (m streamClosedMsg) BlockID() string { return m.blockID }

// NewStreamClosedMsg es el constructor público para el mensaje de cierre.
func NewStreamClosedMsg(id string, err error) tea.Msg {
	return streamClosedMsg{
		blockID: id,
		err:     err,
	}
}