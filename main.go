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

	//"github.com/charmbracelet/bubbles/spinner"
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


type model struct {
	blocks 		[]block.Block
	width  		int
	height 		int
    currentView string // "dashboard" o "viewport"
    viewport    viewport.Model
    //activeBlock 		block.Block // Para saber qué bloque está activo, ya no se usa	
    focusIndex  		int    // Índice del bloque que tiene el foco	
	dashboardVP 		viewport.Model // Para la vista principal
	expandedVP  		viewport.Model  // Para la vista de un solo bloque
	normalBorderStyle 	lipgloss.Style
	focusBorderStyle  	lipgloss.Style
}


// Helper para obtener la ruta del archivo de caché (duplicado de shell_command para uso en main)
func getCacheFilePath(blockName string) string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", blockName))
}


func (m *model) Init() tea.Cmd {
    // Enviar un mensaje inicial a todos los bloques para que comiencen a cargar
    return func() tea.Msg {
        return block.TriggerUpdateMsg{}
    }
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    logging.Log.Printf("--- MAIN UPDATE RECEIVED MSG TYPE: %T ---", msg)

    var cmds []tea.Cmd
    var cmd tea.Cmd

    // Global handling for expanded view and exiting it
    if m.currentView == "expanded" { // antes "viewport"
        if keyMsg, ok := msg.(tea.KeyMsg); ok {
            k := keyMsg.String(); 
            if k == "q" || k == "esc" || k == "enter" {
                m.currentView = "dashboard"
               	return m, nil
            }
        }
        // Pass all other messages to the expanded viewport
		m.expandedVP, cmd = m.expandedVP.Update(msg)
        return m, cmd
    }

	// Si estamos en el dashboard, manejamos los eventos del dashboard.
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
        m.dashboardVP.Width = msg.Width
        m.dashboardVP.Height = msg.Height
		m.expandedVP.Width = msg.Width 
		m.expandedVP.Height = msg.Height
		return m, nil // Termina aquí

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			//if len(m.blocks) > 0 {
			m.currentView = "expanded" // antes "viewport"
			focusedBlock := m.blocks[m.focusIndex]
			m.expandedVP.SetContent(focusedBlock.View())
			m.expandedVP.GotoTop()
			//}
			return m, nil
        // Ahora, las flechas arriba/abajo controlan el desplazamiento del viewport principal.
        case "up", "k":
            m.dashboardVP, cmd = m.dashboardVP.Update(msg) // Pasa el mensaje al viewport principal
            cmds = append(cmds, cmd)
        case "down", "j":
            m.dashboardVP, cmd = m.dashboardVP.Update(msg) // Pasa el mensaje al viewport principal
            cmds = append(cmds, cmd)
        case "pgup", "pgdown": // También para Page Up/Page Down
            m.dashboardVP, cmd = m.dashboardVP.Update(msg)
            cmds = append(cmds, cmd)
        case "tab": // El Tab sigue para cambiar el foco visual (borde)
            m.focusIndex = (m.focusIndex + 1) % len(m.blocks)
            return m, nil 
        }

    

    default:
	    // --- BUCLE DE DELEGACIÓN ---
		// ¿Es un mensaje dirigido a un bloque específico?
		if targetMsg, ok := msg.(block.TargetedMsg); ok {
			// Sí, es dirigido. Buscamos el bloque objetivo.
			targetID := targetMsg.BlockID()
		    for i, b := range m.blocks {
				if b.Name() == targetID {
					updatedBlock, cmd := b.Update(msg)
		        	m.blocks[i] = updatedBlock 
		        	cmds = append(cmds, cmd)
		        	break // out of tha loop
		        }
		    }
	    } else {
			// No, es un mensaje general (como TriggerUpdateMsg o spinner.TickMsg).
			// Lo enviamos a todos los bloques.
			for i, b := range m.blocks {
				updatedBlock, cmd := b.Update(msg)
				m.blocks[i] = updatedBlock
				cmds = append(cmds, cmd)
			}	    	
	    }
	}

	return m, tea.Batch(cmds...)
}

// renderDashboardContent era una función auxiliar que generaba el string
// completo de todos los bloques, listo para ser puesto en el viewport. 
// ya no hace falta, todo va en el View.
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

    		logging.Log.Printf("--- nombre de bloque: %T ---Indice de foco", b.Name())

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
		// Si estamos en vista expandida, mostramos ESE viewport
		return m.expandedVP.View()
	}
    
    if m.width == 0 { return "Initializing..." }

    var finalLayout []string       // Almacenará los elementos finales (columnas unidas y bloques full)
    var leftColumnViews []string   // Vistas pendientes para la columna izquierda
    var rightColumnViews []string  // Vistas pendientes para la columna derecha

    // Función helper para procesar y limpiar las columnas pendientes.
    processPendingColumns := func() {
        if len(leftColumnViews) > 0 || len(rightColumnViews) > 0 {
            leftColumn := lipgloss.JoinVertical(lipgloss.Left, leftColumnViews...)
            rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightColumnViews...)
            
            // Une las columnas horizontalmente
            joinedCols := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
            finalLayout = append(finalLayout, joinedCols)

            // Limpia los slices para la siguiente sección de columnas
            leftColumnViews = []string{}
            rightColumnViews = []string{}
        }
    }

    // 1. Renderizar y agrupar vistas por posición
    for i, b := range m.blocks {
        blockView := b.View()

    	//logging.Log.Printf("FOCUS CHECK: focusIndex=%d, blockIndex=%d, blockName=%s", m.focusIndex, i, b.Name())

        // Aplicar borde y foco (como ya haces)
	    var borderStyle lipgloss.Style
	    if i == m.focusIndex {
	        borderStyle = m.focusBorderStyle
	    } else {
	        borderStyle = m.normalBorderStyle
	    }

		//linea clave para asignar el borde correcto
		//renderedBlock := borderStyle.Width(blockWidth).Render(blockView)
        
        // Asumimos una nueva propiedad `Position` en la interfaz Block.
        // Los bloques sin "left" o "right" se consideran "full".
        position := b.Position() 

        if position == "left" || position == "right" {
            // Renderizamos el bloque con un ancho aproximado de la mitad de la pantalla
            // Menos espacio para padding y bordes.
            blockWidth := (m.width / 2) - 4 
            renderedBlock := borderStyle.Width(blockWidth).Render(blockView)

            if position == "left" {
                leftColumnViews = append(leftColumnViews, renderedBlock)
            } else {
                rightColumnViews = append(rightColumnViews, renderedBlock)
            }
        } else { // El bloque es "full-width"
            // 1. Procesa cualquier columna que estuviera pendiente antes de este bloque.
            processPendingColumns()

            // 2. Renderiza el bloque de ancho completo y añádelo al layout final.
            blockWidth := m.width - 2 // Ancho completo menos bordes
            renderedBlock := borderStyle.Width(blockWidth).Render(blockView)
            finalLayout = append(finalLayout, renderedBlock)
        }

    }

    // 3. Procesa cualquier columna que haya quedado al final del bucle.
    processPendingColumns()

    // Antes devolvíamos todo a lipgloss pero necesitamos el viewport
    // 4A. Unía todos los elementos del layout final verticalmente.
    // return lipgloss.JoinVertical(lipgloss.Left, finalLayout...)

    // 4B. Al viewport
    // 1. Unimos todos los elementos del layout final en un solo string.
    fullLayout := lipgloss.JoinVertical(lipgloss.Left, finalLayout...)

    // 2. Establecemos ese string como el contenido de nuestro viewport.
    m.dashboardVP.SetContent(fullLayout)

    // 3. Devolvemos la vista del viewport, que se encargará del scroll.
    return m.dashboardVP.View()
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

	// LEEMOS ESTILOS UNA SOLA VEZ ---
	normalBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Border)) // Usamos el color del tema

	focusBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Colors.Primary)) // Usamos el color primario para el foco


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
	dashVP := viewport.New(100, 20) // El tamaño inicial no es crítico, se ajusta luego
	expVP := viewport.New(100, 20)
	expVP.Style = baseStyle
	dashVP.Style = baseStyle
	m := &model{
		blocks:      activeBlocks,
		currentView: "dashboard",
		dashboardVP: dashVP,
		expandedVP:  expVP,
		focusIndex:  0, // Inicia el foco en el primer bloque
		normalBorderStyle: normalBorderStyle,
		focusBorderStyle:  focusBorderStyle,
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

		// Paso 3.1: Iniciar la actualización.
		updatedBlock, cmd := b.Update(block.TriggerUpdateMsg{})

		// Paso 3.2: Ejecutar el comando si es necesario.
		if cmd != nil {
			// Esta llamada es bloqueante: espera a que el comando termine y devuelve el mensaje.
			msg := cmd()
			
			// Paso 3.3: Finalizar la actualización.
			updatedBlock, _ = updatedBlock.Update(msg)
		}

		// Paso 4: Imprimir la vista.
		// Ahora que el bloque está completamente actualizado (desde caché o ejecución),
		// imprimimos su vista directamente a la consola.
		fmt.Println(updatedBlock.View())
		fmt.Println("---") // Añadimos un separador simple
	}
}
