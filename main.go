// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/gas/fancy-welcome/blocks/shell_command"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/shared/block"
	"github.com/gas/fancy-welcome/themes"
)

// --- El código del modo interactivo (Bubble Tea) permanece igual ---

var blockFactory = map[string]func() block.Block{
	"ShellCommand": shell_command.New,
}

type model struct {
	blocks []block.Block
	width  int
	height int
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, b := range m.blocks {
		cmds = append(cmds, b.Update())
	}
	return tea.Batch(cmds...)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	default:
		for _, b := range m.blocks {
			if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
				handler.HandleMsg(msg)
			}
		}
	}
	return m, nil
}

func (m *model) View() string {
	var renderedBlocks []string
	for _, b := range m.blocks {
		renderedBlocks = append(renderedBlocks, b.View())
	}
	return strings.Join(renderedBlocks, "\n\n")
}

// --- La nueva lógica principal en main() ---

func main() {
	// 1. Definir y parsear los argumentos de línea de comandos
	simpleOutput := flag.Bool("simple", false, "Muestra la salida como texto plano sin TUI.")
	flag.Parse()

	// 2. Comprobar si debemos usar el modo simple
	// Se activa si la salida no es una terminal (ej. un pipe) O si se usa el flag --simple.
	isPipe := !isatty.IsTerminal(os.Stdout.Fd())
	if isPipe || *simpleOutput {
		runTtyMode()
	} else {
		runTuiMode()
	}
}

// runInteractiveMode contiene toda la lógica de Bubble Tea que teníamos antes.
func runTuiMode() {
	cfg, err := config.LoadConfig()
	if err != nil { log.Fatalf("Error cargando config: %v", err) }
	theme, err := themes.LoadTheme(cfg.Theme.SelectedTheme)
	if err != nil { log.Fatalf("Error cargando tema: %v", err) }
	baseStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Text))

	var activeBlocks []block.Block
	for _, blockName := range cfg.General.EnabledBlocksOrder {
		blockConfig, _ := cfg.Blocks[blockName].(map[string]interface{})
		
		runMode, _ := blockConfig["run_mode"].(string)
		if runMode == "" {
			runMode = "all"
		}
		// Si el bloque está configurado para ejecutarse solo en modo 'tty', saltamos.
		if runMode == "tty" {
			continue
		}

		blockType, _ := blockConfig["type"].(string)
		if factory, ok := blockFactory[blockType]; ok {
			b := factory()
			blockConfig["name"] = blockName
			if err := b.Init(blockConfig, baseStyle.Copy()); err != nil {
				log.Printf("Error inicializando bloque '%s': %v", blockName, err)
				continue
			}
			activeBlocks = append(activeBlocks, b)
		}
	}
	
	m := &model{blocks: activeBlocks}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error ejecutando el programa: %v\n", err)
		os.Exit(1)
	}
}

// runSimpleMode es la nueva función para la salida de texto plano.
func runTtyMode() {
	cfg, err := config.LoadConfig()
	if err != nil { log.Fatalf("Error cargando config: %v", err) }

	// En modo simple no necesitamos temas visuales.
	// Creamos los bloques sin estilo.
	var activeBlocks []block.Block
	for _, blockName := range cfg.General.EnabledBlocksOrder {
		blockConfig, _ := cfg.Blocks[blockName].(map[string]interface{})
		
		runMode, _ := blockConfig["run_mode"].(string)
		if runMode == "" {
			runMode = "all"
		}
		// Si el bloque está configurado para ejecutarse solo en modo 'tui', saltamos.
		if runMode == "tui" {
			continue
		}

		blockType, _ := blockConfig["type"].(string)
		if factory, ok := blockFactory[blockType]; ok {
			b := factory()
			blockConfig["name"] = blockName
			// Pasamos un estilo vacío que no hará nada.
			if err := b.Init(blockConfig, lipgloss.NewStyle()); err != nil {
				continue
			}
			activeBlocks = append(activeBlocks, b)
		}
	}

	// 3. Ejecutar los bloques de forma síncrona
	for _, b := range activeBlocks {
		cmd := b.Update()
		if cmd == nil {
			continue
		}
		// Ejecutamos el comando y obtenemos el mensaje directamente.
		msg := cmd()
		
		// Pasamos el mensaje al bloque para que actualice su estado interno.
		if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
			handler.HandleMsg(msg)
		}

		// Imprimimos la vista del bloque directamente a la consola.
		fmt.Println(b.View())
		fmt.Println("---") // Añadimos un separador simple
	}
}
