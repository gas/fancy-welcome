// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"github.com/gas/fancy-welcome/blocks/shell_command"
	"github.com/gas/fancy-welcome/blocks/system_info"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/shared/block"
	"github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/logging"
)

// --- El código del modo interactivo (Bubble Tea) permanece igual ---

var blockFactory = map[string]func() block.Block{
	"ShellCommand": shell_command.New,
	"SystemInfo":   system_info.New,
}

func periodicUpdate() tea.Cmd {
	// El tick global ahora es más rápido, la lógica de frecuencia está en cada bloque.
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return periodicUpdateMsg(t)
	})
}

type model struct {
	blocks []block.Block
	width  int
	height int
    currentView string // "dashboard" o "viewport"
    viewport    viewport.Model
    //activeBlock block.Block // Para saber qué bloque está activo, puede ser muy útil	
    focusIndex  int    // Índice del bloque que tiene el foco	
}


// Helper para obtener la ruta del archivo de caché (duplicado de shell_command para uso en main)
func getCacheFilePath(blockName string) string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", blockName))
}


type periodicUpdateMsg time.Time

func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, b := range m.blocks {
		////cmds = append(cmds, b.Update())
		// Recogemos tanto el comando de actualización de datos...
		cmds = append(cmds, b.Update()) 
		// ...como el comando para iniciar la animación del spinner, si existe.
		if s, ok := b.(interface{ SpinnerCmd() tea.Cmd }); ok {
			cmds = append(cmds, s.SpinnerCmd())
		}
	}

	cmds = append(cmds, periodicUpdate())

    // Asumiendo que al menos un bloque tiene spinner para iniciar el tick.
    // Una implementación más robusta comprobaría esto.
	//if len(m.blocks) > 0 {
	//	if s, ok := m.blocks[0].(interface{ SpinnerCmd() tea.Cmd }); ok {
	//		cmds = append(cmds, s.SpinnerCmd())
	//	}
	//}

	return tea.Batch(cmds...)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd

    // Global handling for exiting expanded view
    if m.currentView == "expanded" { // antes "viewport"
        //switch msg := msg.(type) {
        //case tea.KeyMsg:
		//	if k := msg.String(); k == "q" || k == "esc" {
		//		m.currentView = "dashboard"
		//		return m, nil
		//	}
        //}
        if msg, ok := msg.(tea.KeyMsg); ok {
            if k := msg.String(); k == "q" || k == "esc" {
                m.currentView = "dashboard"
                return m, nil
            }
        }
        // Pass all other messages to the viewport
        m.viewport, cmd = m.viewport.Update(msg)
        return m, cmd
    }

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height

        // debes recalcular y establecer el contenido completo del viewport principal
        // para que pueda determinar si necesita desplazarse.
        m.viewport.SetContent(m.renderDashboardContent()) // ¡Nueva función!

		for _, b := range m.blocks {
			if s, ok := b.(interface{ SetWidth(int) }); ok {
				s.SetWidth(m.width)
			}
		}
		return m, nil

	case periodicUpdateMsg:
		// LOGGING: Añadimos esta línea para ver cada tick en el log.
		logging.Log.Println("\n--- Periodic Update Tick Received ---")
	
		for _, b := range m.blocks {
			// This will trigger an update, which will be ignored if cache is still valid
			cmds = append(cmds, b.Update())
		}
		//return m, tea.Batch(cmds..., periodicUpdate()) // así no.
		cmds = append(cmds, periodicUpdate()) // así sí?
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if len(m.blocks) > 0 {
				m.currentView = "expanded" // antes "viewport"
				// Usa el bloque con foco para la vista de viewport
				//m.activeBlock = m.blocks[m.focusIndex]
				//m.viewport.SetContent(m.activeBlock.View())
				//m.viewport.GotoTop()
				focusedBlock := m.blocks[m.focusIndex]
				m.viewport.SetContent(focusedBlock.View())
				m.viewport.GotoTop()
				return m, nil
			}
        // Ahora, las flechas arriba/abajo controlan el desplazamiento del viewport principal.
        case "up", "k":
            m.viewport, cmd = m.viewport.Update(msg) // Pasa el mensaje al viewport principal
            cmds = append(cmds, cmd)
        case "down", "j":
            m.viewport, cmd = m.viewport.Update(msg) // Pasa el mensaje al viewport principal
            cmds = append(cmds, cmd)
        case "pgup", "pgdown": // También para Page Up/Page Down
            m.viewport, cmd = m.viewport.Update(msg)
            cmds = append(cmds, cmd)
        case "tab": // El Tab sigue para cambiar el foco visual (borde)
            m.focusIndex = (m.focusIndex + 1) % len(m.blocks)
            // Después de cambiar el foco, puedes querer asegurarte de que el panel enfocado
            // esté visible en el viewport. Puedes llamar a m.viewport.SetYOffset(offset)
            // para desplazarlo. Esto es un poco más complejo y lo veremos más adelante si es necesario.
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
		// Pass other messages (like dataMsg) to all blocks
		for _, b := range m.blocks {
			if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
				handler.HandleMsg(msg)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

// renderDashboardContent es una nueva función auxiliar que genera el string
// completo de todos los bloques, listo para ser puesto en el viewport.
func (m *model) renderDashboardContent() string {
    if m.width == 0 {
        return "Initializing..."
    }

    var renderedBlocks []string
    for i, b := range m.blocks {
        blockView := b.View()

        // Estilo para el borde del bloque
        borderStyle := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(lipgloss.Color("240")) // Color de borde por defecto

        // Si el bloque tiene el foco, cambia el color del borde
        if i == m.focusIndex {
            borderStyle = borderStyle.BorderForeground(lipgloss.Color("12")) // Color de borde para el foco
        }

        // Aplicamos el ancho al estilo y luego renderizamos el contenido DENTRO del borde.
        renderedBlocks = append(renderedBlocks, borderStyle.Width(m.width - 2).Render(blockView))
    }
    return strings.Join(renderedBlocks, "\n")
}

func (m *model) View() string {
    if m.currentView == "expanded" {
    	// en vista expandida solo viewport de ese bloque
        return m.viewport.View()
    }	

    // En el modo dashboard, el viewport principal contiene todo el contenido renderizado.
    // Asegúrate de que el viewport tiene el contenido más reciente.
    // Esto es crucial para que el desplazamiento funcione correctamente.
    m.viewport.SetContent(m.renderDashboardContent())
    return m.viewport.View()

}


// --- La nueva lógica principal en main() ---

func main() {
	// Inicializamos el logger al principio de todo.
	logFile, err := logging.Init()
	if err != nil {
		// Si no podemos crear el log, al menos lo notificamos por la consola.
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	// Nos aseguramos de que el archivo de log se cierre al salir de la aplicación.
	defer logFile.Close()

	// 1. Definir y parsear los argumentos de línea de comandos
	simpleOutput := flag.Bool("simple", false, "Muestra la salida como texto plano sin TUI.")
	refreshFlag := flag.String("refresh", "", "Forzar el refresco de un bloque específico o 'all' para todos.")
	flag.Parse()

	// 2. Comprobar si debemos usar el modo simple
	// Se activa si la salida no es una terminal (ej. un pipe) O si se usa el flag --simple.
	isPipe := !isatty.IsTerminal(os.Stdout.Fd())
	if isPipe || *simpleOutput {
		runTtyMode(*refreshFlag)
	} else {
		runTuiMode(*refreshFlag)
	}

}

// runInteractiveMode contiene toda la lógica de Bubble Tea que teníamos antes.
func runTuiMode(refreshTarget string) {
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

		// Forzar refresco si es el objetivo
		if refreshTarget == "all" || refreshTarget == blockName {
			cacheFile := getCacheFilePath(blockName)
			_ = os.Remove(cacheFile) // Ignoramos el error si el archivo no existe
		}

		blockConfig, _ := cfg.Blocks[blockName].(map[string]interface{})	
		runMode, _ := blockConfig["run_mode"].(string)
		if runMode == "" { runMode = "all" }
		// Si el bloque está configurado para ejecutarse solo en modo 'tty', saltamos.
		if runMode == "tty" { continue }

		blockType, _ := blockConfig["type"].(string)
		if factory, ok := blockFactory[blockType]; ok {
			b := factory()
			blockConfig["name"] = blockName
			if err := b.Init(blockConfig, cfg.General, theme); err != nil {
				log.Printf("Error inicializando bloque '%s': %v", blockName, err)
				continue
			}
			activeBlocks = append(activeBlocks, b)
		}
	}

	// 3: La inicialización del modelo se hace aquí, dentro del modo TUI.
	vp := viewport.New(100, 20)
	vp.Style = baseStyle
	m := &model{
		blocks:      activeBlocks,
		currentView: "dashboard",
		viewport:    vp,
		focusIndex:  0, // Inicia el foco en el primer bloque
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error ejecutando el programa: %v\n", err)
		os.Exit(1)
	}
}

// runSimpleMode es la nueva función para la salida de texto plano.
func runTtyMode(refreshTarget string) {
	//config
	cfg, err := config.LoadConfig()
	if err != nil { 
		logging.Log.Fatalf("Error cargando config: %v", err) 
	}

	//themes
	theme, err := themes.LoadTheme(cfg.Theme.SelectedTheme)
	if err != nil { log.Fatalf("Error cargando tema: %v", err) }

	//cache
	homeDir, err := os.UserHomeDir()
	if err != nil { 
		logging.Log.Fatalf("Error obteniendo home dir: %v", err)
	}
	cacheDir := filepath.Join(homeDir, ".cache", "fancy-welcome")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
	    logging.Log.Fatalf("Error creando directorio de caché: %v", err)
	}

	// En modo tty no necesitamos temas visuales.
	// Creamos los bloques sin estilo.
	var activeBlocks []block.Block
	for _, blockName := range cfg.General.EnabledBlocksOrder {
		// Forzar refresco si es el objetivo
		if refreshTarget == "all" || refreshTarget == blockName {
			cacheFile := getCacheFilePath(blockName)
			_ = os.Remove(cacheFile) // Ignoramos el error si el archivo no existe
		}

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

			if err := b.Init(blockConfig, cfg.General, theme); err != nil {
				logging.Log.Printf("Error inicializando bloque '%s': %v", blockName, err)
				continue
			}
			activeBlocks = append(activeBlocks, b)
		}
	}

	// 3. Ejecutar los bloques de forma síncrona
	for _, b := range activeBlocks {
		cmd := b.Update()

		// Si se devolvió un comando (no se usó la caché), ejecútalo.
		if cmd != nil {
			// Ejecutamos el comando y obtenemos el mensaje directamente.
			msg := cmd()
			
			// Pasamos el mensaje al bloque para que actualice su estado interno.
			if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
				handler.HandleMsg(msg)
			}
		}

		// Ahora, el estado del bloque está actualizado (desde caché o ejecución).
		// Imprimimos la vista del bloque directamente a la consola.
		fmt.Println(b.View())
		fmt.Println("---") // Añadimos un separador simple
	}
}
