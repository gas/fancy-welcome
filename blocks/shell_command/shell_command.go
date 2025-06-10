// blocks/shell_command/shell_command.go
package shell_command

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
    // Asegúrate de que las rutas de importación coincidan con tu módulo
	"github.com/gas/fancy-welcome/blocks/shell_command/parsers"
	"github.com/gas/fancy-welcome/blocks/shell_command/renderers"
	"github.com/gas/fancy-welcome/shared/block"
)

var registeredParsers = make(map[string]parsers.Parser)
var registeredRenderers = make(map[string]renderers.Renderer)

func init() {
	registeredParsers["single_line"] = &parsers.SingleLineParser{}
	registeredParsers["multi_line"] = &parsers.MultiLineParser{}
	
	registeredRenderers["raw_text"] = &renderers.RawTextRenderer{}
	registeredRenderers["cowsay"] = &renderers.CowsayRenderer{}
}

// CORRECCIÓN 1: Añadir el campo 'id' al struct del bloque.
type ShellCommandBlock struct {
	id           string // ID único del bloque
	style        lipgloss.Style
	command      string
	args         []string
	parser       parsers.Parser
	renderer     renderers.Renderer
	parsedData   interface{}
	currentError error
}

// CORRECCIÓN 2: Añadir 'blockID' al mensaje para saber a quién pertenece.
type dataMsg struct {
	blockID string
	data    interface{}
	err     error
}

func New() block.Block {
	return &ShellCommandBlock{}
}

func (b *ShellCommandBlock) Name() string {
    // Usamos el id como el nombre, ya que es único.
	return b.id
}

func (b *ShellCommandBlock) Init(config map[string]interface{}, style lipgloss.Style) error {
	// CORRECCIÓN 3: Guardamos el nombre único del bloque como su ID.
	b.id = config["name"].(string) 
	b.style = style

	cmdRaw, _ := config["command"].(string)
	if cmdRaw == "" {
		return fmt.Errorf("el campo 'command' es obligatorio para el bloque '%s'", b.id)
	}
	parts := strings.Fields(cmdRaw)
	b.command = parts[0]
	if len(parts) > 1 {
		b.args = parts[1:]
	}

	parserName, _ := config["parser"].(string)
	p, ok := registeredParsers[parserName]
	if !ok {
		return fmt.Errorf("parser '%s' no encontrado para el bloque '%s'", parserName, b.id)
	}
	b.parser = p

	rendererName, _ := config["renderer"].(string)
	r, ok := registeredRenderers[rendererName]
	if !ok {
		return fmt.Errorf("renderer '%s' no encontrado para el bloque '%s'", rendererName, b.id)
	}
	b.renderer = r
	
	return nil
}

func (b *ShellCommandBlock) Update() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(b.command, b.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
            // CORRECCIÓN 4 (Parte A): Incluimos el ID en el mensaje de error.
			return dataMsg{blockID: b.id, err: fmt.Errorf("falló la ejecución del comando: %w", err)}
		}

		parsedData, err := b.parser.Parse(string(output))
		if err != nil {
            // CORRECCIÓN 4 (Parte B): Incluimos el ID en el mensaje de error.
			return dataMsg{blockID: b.id, err: fmt.Errorf("falló el parseo: %w", err)}
		}
		
        // CORRECCIÓN 4 (Parte C): Incluimos el ID en el mensaje de éxito.
		return dataMsg{blockID: b.id, data: parsedData}
	}
}

func (b *ShellCommandBlock) View() string {
	if b.currentError != nil {
		return b.style.Render(fmt.Sprintf("Error en '%s': %v", b.id, b.currentError))
	}
	if b.parsedData == nil {
		// para soportar salida tty por ahora eliminamos el cargando
		// más adelante pensar implementar algo como el spin de charm gum
		// return b.style.Render(fmt.Sprintf("Cargando '%s'...", b.id))
		return ""
	}
	return b.renderer.Render(b.parsedData, 0, b.style)
}

func (b *ShellCommandBlock) HandleMsg(msg tea.Msg) {
    // CORRECCIÓN 5: Comprobamos si el mensaje es un dataMsg Y si su ID coincide con el nuestro.
	if m, ok := msg.(dataMsg); ok && m.blockID == b.id {
		b.parsedData = m.data
		b.currentError = m.err
	}
}
