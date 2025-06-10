// blocks/shell_command/shell_command.go
package shell_command

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
    	"github.com/gas/fancy-welcome/blocks/shell_command/parsers"
	"github.com/gas/fancy-welcome/blocks/shell_command/renderers"
	"github.com/gas/fancy-welcome/shared/block"
)

// Registros para nuestros parsers y renderers
var registeredParsers = make(map[string]parsers.Parser)
var registeredRenderers = make(map[string]renderers.Renderer)

// init se ejecuta una vez al inicio del programa para poblar los registros.
func init() {
	registeredParsers["single_line"] = &parsers.SingleLineParser{}
	registeredParsers["multi_line"] = &parsers.MultiLineParser{}
	
	registeredRenderers["raw_text"] = &renderers.RawTextRenderer{}
	registeredRenderers["cowsay"] = &renderers.CowsayRenderer{}
}

// ShellCommandBlock es la implementación del bloque.
type ShellCommandBlock struct {
	name         string
	style        lipgloss.Style
	command      string
	args         []string
	parser       parsers.Parser
	renderer     renderers.Renderer
	parsedData   interface{}
	currentError error
}

// New crea una nueva instancia del bloque.
func New() block.Block {
	return &ShellCommandBlock{}
}

func (b *ShellCommandBlock) Name() string {
	return b.name
}

func (b *ShellCommandBlock) Init(config map[string]interface{}, style lipgloss.Style) error {
    b.name = config["name"].(string) // Asumimos que el nombre viene en la config
	b.style = style

	// Extraer y validar configuración
	cmdRaw, _ := config["command"].(string)
	if cmdRaw == "" {
		return fmt.Errorf("el campo 'command' es obligatorio para el bloque '%s'", b.name)
	}
	parts := strings.Fields(cmdRaw)
	b.command = parts[0]
	if len(parts) > 1 {
		b.args = parts[1:]
	}

	parserName, _ := config["parser"].(string)
	p, ok := registeredParsers[parserName]
	if !ok {
		return fmt.Errorf("parser '%s' no encontrado para el bloque '%s'", parserName, b.name)
	}
	b.parser = p

	rendererName, _ := config["renderer"].(string)
	r, ok := registeredRenderers[rendererName]
	if !ok {
		return fmt.Errorf("renderer '%s' no encontrado para el bloque '%s'", rendererName, b.name)
	}
	b.renderer = r
	
	return nil
}

type dataMsg struct {
	data  interface{}
	err   error
}

func (b *ShellCommandBlock) Update() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(b.command, b.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return dataMsg{err: fmt.Errorf("falló la ejecución del comando: %w", err)}
		}

		parsedData, err := b.parser.Parse(string(output))
		if err != nil {
			return dataMsg{err: fmt.Errorf("falló el parseo: %w", err)}
		}
		
		return dataMsg{data: parsedData}
	}
}

func (b *ShellCommandBlock) View() string {
	if b.currentError != nil {
		return b.style.Render(fmt.Sprintf("Error en '%s': %v", b.name, b.currentError))
	}
	if b.parsedData == nil {
		return b.style.Render("Cargando...")
	}
	// Pasamos el ancho (width) como 0 por ahora, los renderers aún no lo usan.
	return b.renderer.Render(b.parsedData, 0, b.style)
}

// HandleMsg es un método para que el modelo principal le pase mensajes al bloque.
func (b *ShellCommandBlock) HandleMsg(msg tea.Msg) {
    if m, ok := msg.(dataMsg); ok {
        b.parsedData = m.data
        b.currentError = m.err
    }
}
