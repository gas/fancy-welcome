// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"path/filepath"
	//"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
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
    currentView string // "dashboard" o "viewport"
    viewport    viewport.Model
    activeBlock block.Block // Para saber qué bloque está activo	
}

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, b := range m.blocks {
		cmds = append(cmds, b.Update())
	}

    // Asumiendo que al menos un bloque tiene spinner para iniciar el tick.
    // Una implementación más robusta comprobaría esto.
	if len(m.blocks) > 0 {
		if s, ok := m.blocks[0].(interface{ SpinnerCmd() tea.Cmd }); ok {
			cmds = append(cmds, s.SpinnerCmd())
		}
	}

	return tea.Batch(cmds...)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd

    // Si estamos en la vista del viewport, le pasamos los mensajes
    if m.currentView == "viewport" {
        switch msg := msg.(type) {
        case tea.KeyMsg:
			if k := msg.String(); k == "q" || k == "esc" {
				m.currentView = "dashboard"
				return m, nil
			}
        }
        m.viewport, cmd = m.viewport.Update(msg)
        return m, cmd
    }

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if k := msg.String(); k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
		if k := msg.String(); k == "v" && len(m.blocks) > 0 {
			m.currentView = "viewport"
			m.activeBlock = m.blocks[0]
			m.viewport.SetContent(m.activeBlock.View())
			m.viewport.GotoTop()
			return m, nil
		}

	//1: El spinner se anima con tea.TickMsg.?? seguro?
	//case tea.TickMsg:
	case spinner.TickMsg:
		for i := range m.blocks {
			if s, ok := m.blocks[i].(interface{ Spinner() *spinner.Model }); ok {
				var spinnerCmd tea.Cmd
				*s.Spinner(), spinnerCmd = s.Spinner().Update(msg)
				cmds = append(cmds, spinnerCmd)
			}
		}
		return m, tea.Batch(cmds...)

	default:
		for _, b := range m.blocks {
			if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
				handler.HandleMsg(msg)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *model) View() string {
    if m.currentView == "viewport" {
        return m.viewport.View()
    }	
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
	//config
	cfg, err := config.LoadConfig()
	if err != nil { log.Fatalf("Error cargando config: %v", err) }
	//themes
	theme, err := themes.LoadTheme(cfg.Theme.SelectedTheme)
	if err != nil { log.Fatalf("Error cargando tema: %v", err) }
	baseStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Text))
	//cache
	homeDir, err := os.UserHomeDir()
	if err != nil { log.Fatalf("Error obteniendo home dir: %v", err) }
	cacheDir := filepath.Join(homeDir, ".cache", "fancy-welcome")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
	    log.Fatalf("Error creando directorio de caché: %v", err)
	}

	var activeBlocks []block.Block
	for _, blockName := range cfg.General.EnabledBlocksOrder {
		blockConfig, _ := cfg.Blocks[blockName].(map[string]interface{})	
		runMode, _ := blockConfig["run_mode"].(string)
		if runMode == "" { runMode = "all" }
		// Si el bloque está configurado para ejecutarse solo en modo 'tty', saltamos.
		if runMode == "tty" { continue }

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

	// 3: La inicialización del modelo se hace aquí, dentro del modo TUI.
	vp := viewport.New(100, 20)
	m := &model{
		blocks:      activeBlocks,
		currentView: "dashboard",
		viewport:    vp,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error ejecutando el programa: %v\n", err)
		os.Exit(1)
	}
}

// runSimpleMode es la nueva función para la salida de texto plano.
func runTtyMode() {
	//config
	cfg, err := config.LoadConfig()
	if err != nil { log.Fatalf("Error cargando config: %v", err) }

	//cache
	homeDir, err := os.UserHomeDir()
	if err != nil { log.Fatalf("Error obteniendo home dir: %v", err) }
	cacheDir := filepath.Join(homeDir, ".cache", "fancy-welcome")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
	    log.Fatalf("Error creando directorio de caché: %v", err)
	}
	// En modo simple no necesitamos temas visuales.
	// Creamos los bloques sin estilo.
	var activeBlocks []block.Block
	for _, blockName := range cfg.General.EnabledBlocksOrder {
		blockConfig, _ := cfg.Blocks[blockName].(map[string]interface{})
		
		runMode, _ := blockConfig["run_mode"].(string)
		if runMode == "" { runMode = "all" }
		// Si el bloque está configurado para ejecutarse solo en modo 'tui', saltamos.
		if runMode == "tui" { continue }

		blockType, _ := blockConfig["type"].(string)
		if factory, ok := blockFactory[blockType]; ok {
			b := factory()
			blockConfig["name"] = blockName
			// Pasamos un estilo vacío que no hará nada.
			if err := b.Init(blockConfig, lipgloss.NewStyle()); err != nil { continue }
			activeBlocks = append(activeBlocks, b)
		}
	}

	// 3. Ejecutar los bloques de forma síncrona
	for _, b := range activeBlocks {
		cmd := b.Update()
		if cmd == nil { continue }
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
