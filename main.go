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
    "github.com/gas/fancy-welcome/shared/messages"
    "github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/logging"
)

// --- El código del modo interactivo (Bubble Tea) permanece igual ---

var blockFactory = map[string]func() block.Block{
	"ShellCommand": shell_command.New,
	"SystemInfo":   system_info.New,
}



type model struct {
	blocks []block.Block
	width  int
	height int
    currentView string // "dashboard" o "viewport"
    viewport    viewport.Model
    //activeBlock block.Block // Para saber qué bloque está activo, puede ser muy útil	
    focusIndex  int    // Índice del bloque que tiene el foco	
	focusableBlocksOrder []block.Block // Lista que representa el orden visual de los bloques para el foco
	dashboardNeedsRender bool          // Nuevo flag: true si el contenido del dashboard (o su layout/altura) ha cambiado
    // Nueva estructura para almacenar información del layout visual
    blockLayoutInfo []struct {
        Block     block.Block
        StartLine int // Línea de inicio de este bloque en el dashboard total
        Height    int // Altura total de este bloque renderizado
    }
    spinnerForTicks spinner.Model // Un spinner "invisible" solo para generar ticks
}


// Helper para obtener la ruta del archivo de caché (duplicado de shell_command para uso en main)
func getCacheFilePath(blockName string) string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", blockName))
}


func (m *model) Init() tea.Cmd {
	var cmds []tea.Cmd

    // **** CAMBIO AQUI ****
    // Inicializar el spinner "invisible" con un tipo de spinner conocido y probado.
    // Esto es para aislar si el problema está en la inicialización del spinner o en el flujo de mensajes.
    m.spinnerForTicks = spinner.New(
        spinner.WithSpinner(spinner.Spinner{ // <-- Esta es la forma correcta de usar WithSpinner
            Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, // Los frames por defecto
            FPS: time.Second / 10, // 10 FPS
        }),
    )

    cmds = append(cmds, m.spinnerForTicks.Tick) // Inicia el comando para que este spinner genere ticks
    logging.Log.Println("DEBUG: spinnerForGlobalTicks.Tick command added to initial batch with known spinner.")
    // ********************

//    m.spinnerForTicks = spinner.New(
//        spinner.WithSpinner(spinner.Spinner{ // Usa el spinner por defecto o uno de tus temas
//            Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}, // Default spinner frames
//            FPS: time.Second / 10, // 10 fotogramas por segundo (o lo que desees)
//        }),
//    )
//    cmds = append(cmds, m.spinnerForTicks.Tick) // Inicia el comando para que este spinner genere ticks


	for _, b := range m.blocks {
		cmds = append(cmds, b.Update()) 
	}

	//cmds = append(cmds, periodicUpdate())
    m.dashboardNeedsRender = true // Asegúrate de que el dashboard se renderiza al inicio
	m.updateFocusableOrder()

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

		m.dashboardNeedsRender = true // La ventana cambió de tamaño, hay que re-renderizar todo

        // debes recalcular y establecer el contenido completo del viewport principal
        // para que pueda determinar si necesita desplazarse.
        m.viewport.SetContent(m.renderDashboardContent()) // ¡Nueva función!

		for _, b := range m.blocks {
			if s, ok := b.(interface{ SetWidth(int) }); ok {
				s.SetWidth(m.width)
			}
		}
		return m, nil

	// nuevo case
    case messages.FreshDataMsg, messages.CachedDataMsg: // Interceptamos mensajes de datos para marcar `dashboardNeedsRender`
        for _, b := range m.blocks {
            if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
                handler.HandleMsg(msg) // Pasamos el mensaje al bloque para que actualice su estado
                // Si el bloque indica que su contenido ha cambiado, marcamos el dashboard para re-renderizar
                if b.HasContentChanged() {
                    m.dashboardNeedsRender = true
                }
            }
        }
        return m, tea.Batch(cmds...) // Continúa procesando posibles comandos generados por HandleMsg


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
            if len(m.focusableBlocksOrder) > 0 {	            
                m.focusIndex = (m.focusIndex + 1) % len(m.focusableBlocksOrder) 
            	m.dashboardNeedsRender = true // El foco cambió, el render necesita actualizar los bordes
            	// Lógica de desplazamiento automático (¡esto puede ser costoso!)
            	m.adjustViewportToFocus() // Nueva función auxiliar para el desplazamiento
	        }
            return m, nil
	    }

	//1: El spinner se anima con tea.TickMsg.?? seguro?
	//case tea.TickMsg:
	case spinner.TickMsg:
    	//logging.Log.Println("DEBUG: Received spinner.TickMsg.")
    	var spinnerCmd tea.Cmd

	    m.spinnerForTicks, spinnerCmd = m.spinnerForTicks.Update(msg)
	    cmds = append(cmds, spinnerCmd) 
    	//logging.Log.Println(fmt.Sprintf("DEBUG: Reprogramming next spinner tick. Cmd is nil: %t", spinnerCmd == nil))

		for i := range m.blocks {
			if s, ok := m.blocks[i].(interface{ Spinner() *spinner.Model }); ok {
				*s.Spinner(), _ = s.Spinner().Update(msg) // Solo actualiza
			}
		}

		// 3. ¡NUEVO! Usamos este tick para disparar las actualizaciones de datos.
		// La lógica interna de cada bloque (`nextRunTime`) decidirá si realmente hace algo.
		for _, b := range m.blocks {
			if cmd := b.Update(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	    m.dashboardNeedsRender = true // Un tick del spinner significa que la UI debe re-renderizarse

		return m, tea.Batch(cmds...)

	//default:
		// Pass other messages (like dataMsg) to all blocks
	//	for _, b := range m.blocks {
	//		if handler, ok := b.(interface{ HandleMsg(tea.Msg) }); ok {
	//			handler.HandleMsg(msg)

    //          	if b.HasContentChanged() { // Recheck here if not handled specifically above
    //           	m.dashboardNeedsRender = true
    //          	}
   	//		}
	//	}
	}
	return m, tea.Batch(cmds...)
}

// función auxiliar que genera el string completo de todos los bloques, 
// listo para ser puesto en el viewport.
func (m *model) renderDashboardContent() string {
    if m.width == 0 {
        return "Initializing..."
    }


    finalOutputLines := []string{}
    m.blockLayoutInfo = []struct{ Block block.Block; StartLine int; Height int }{} // Resetear

    currentRenderLine := 0 // Contador para la línea de inicio de cada bloque

    var currentRowBlocks []block.Block // Para acumular bloques que irán en la misma fila horizontal

    // Recorremos los bloques en su orden original de configuración (m.blocks)
    //for blockIndexInConfig, currentBlock := range m.blocks {
    for _, currentBlock := range m.blocks {
        position := currentBlock.GetPosition()

        if position == "left" {
            // Si es "left", lo añadimos a la fila actual y esperamos un "right"
            currentRowBlocks = append(currentRowBlocks, currentBlock)
        } else if position == "right" {
            // Si es "right" y tenemos un "left" esperando, formamos una fila horizontal
            if len(currentRowBlocks) > 0 && currentRowBlocks[0].GetPosition() == "left" {
            	// hay un par left + right renderizamos horizontal
                leftBlock := currentRowBlocks[0]
                rightBlock := currentBlock // Este es el bloque "right" actual

                halfWidth := (m.width / 2) // Divide el ancho disponible por 2
                // Puedes restar 1 para un pequeño espacio/separador si quieres: halfWidth := (m.width / 2) - 1
                if halfWidth < 0 { halfWidth = 0 }

                // Asignar el ancho calculado a cada bloque antes de renderizar
                leftBlock.SetWidth(halfWidth)
                rightBlock.SetWidth(halfWidth)

                // Determinar si alguno de estos bloques tiene el foco
                isLeftBlockFocused := false
                isRightBlockFocused := false
                if len(m.focusableBlocksOrder) > m.focusIndex { // Evitar pánico si focusIndex es inválido
                    focusedBlock := m.focusableBlocksOrder[m.focusIndex]
                    if focusedBlock == leftBlock {
                        isLeftBlockFocused = true
                    } else if focusedBlock == rightBlock {
                        isRightBlockFocused = true
                    }
                }

                // Renderizar cada bloque estilizado y obtener su altura
                renderedLeftStr, heightLeft := m.renderStyledBlock(leftBlock, isLeftBlockFocused)
                renderedRightStr, heightRight := m.renderStyledBlock(rightBlock, isRightBlockFocused)

                // La altura de la fila combinada es la altura máxima de los bloques que la componen.
                rowHeight := max(heightLeft, heightRight)

                // Ajustar la altura de los bloques individuales para que coincidan con la altura de la fila,
                // rellenando con espacios si es necesario (JoinVertical por defecto no lo hace si no se especifica)
                // Usamos AlignVertical(lipgloss.Top) para asegurarnos que el contenido se pega arriba.
                styledLeft := lipgloss.NewStyle().Height(rowHeight).AlignVertical(lipgloss.Top).Render(renderedLeftStr)
                styledRight := lipgloss.NewStyle().Height(rowHeight).AlignVertical(lipgloss.Top).Render(renderedRightStr)

                // Unir los bloques horizontalmente
                combinedRow := lipgloss.JoinHorizontal(lipgloss.Top, styledLeft, styledRight)
                finalOutputLines = append(finalOutputLines, combinedRow)
                
                // Almacenar información de layout para ambos bloques que forman esta fila
                // Asumimos que ambos comparten la misma StartLine pero el 'tab' los diferenciará
                m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
                    Block: leftBlock,
                    StartLine: currentRenderLine,
                    Height: rowHeight,
                })
                 m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
                    Block: rightBlock,
                    StartLine: currentRenderLine,
                    Height: rowHeight,
                })

                currentRenderLine += rowHeight // Avanzar el contador de línea
                currentRowBlocks = nil // Reiniciar para la siguiente fila

            } else {
                // "right" aparece sin un "left" precedente: renderizarlo como "full"
                // Primero, limpia cualquier "left" pendiente si lo hubiera (debería ser handled por el flujo normal)
                for _, pendingBlock := range currentRowBlocks {
                    pendingBlock.SetWidth(m.width) // Se le da ancho completo
                    isPendingBlockFocused := false // Asume que no tiene foco si no fue consumido
                    if len(m.focusableBlocksOrder) > m.focusIndex && m.focusableBlocksOrder[m.focusIndex] == pendingBlock {
                        isPendingBlockFocused = true
                    }
                    renderedStr, renderedHeight := m.renderStyledBlock(pendingBlock, isPendingBlockFocused)
                    finalOutputLines = append(finalOutputLines, renderedStr)
                    m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
                        Block: pendingBlock, StartLine: currentRenderLine, Height: renderedHeight,
                    })
                    currentRenderLine += renderedHeight
                }
                currentRowBlocks = nil

                // Renderizar el actual "right" como "full"
                currentBlock.SetWidth(m.width)
                isCurrentBlockFocused := false
                if len(m.focusableBlocksOrder) > m.focusIndex && m.focusableBlocksOrder[m.focusIndex] == currentBlock {
                    isCurrentBlockFocused = true
                }
                renderedStr, renderedHeight := m.renderStyledBlock(currentBlock, isCurrentBlockFocused)
                finalOutputLines = append(finalOutputLines, renderedStr)
                m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
                    Block: currentBlock, StartLine: currentRenderLine, Height: renderedHeight,
                })
                currentRenderLine += renderedHeight
			}
        } else { // position == "full" o cualquier otro valor (por defecto)
            // Si hay bloques pendientes en currentRowBlocks (un "left" que no encontró su "right"),
            // renderizarlos como "full" antes del bloque actual.
            for _, pendingBlock := range currentRowBlocks {
                pendingBlock.SetWidth(m.width)
                isPendingBlockFocused := false
                 if len(m.focusableBlocksOrder) > m.focusIndex && m.focusableBlocksOrder[m.focusIndex] == pendingBlock {
                    isPendingBlockFocused = true
                }
                renderedStr, renderedHeight := m.renderStyledBlock(pendingBlock, isPendingBlockFocused)
                finalOutputLines = append(finalOutputLines, renderedStr)
                m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
                    Block: pendingBlock, StartLine: currentRenderLine, Height: renderedHeight,
                })
                currentRenderLine += renderedHeight
            }
            currentRowBlocks = nil // Limpiar los pendientes

            // Renderizar el bloque actual (que es "full")
            currentBlock.SetWidth(m.width)
            isCurrentBlockFocused := false
            if len(m.focusableBlocksOrder) > m.focusIndex && m.focusableBlocksOrder[m.focusIndex] == currentBlock {
                isCurrentBlockFocused = true
            }
            renderedStr, renderedHeight := m.renderStyledBlock(currentBlock, isCurrentBlockFocused)
            finalOutputLines = append(finalOutputLines, renderedStr)
            m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
                Block: currentBlock, StartLine: currentRenderLine, Height: renderedHeight,
            })
            currentRenderLine += renderedHeight            
        }
    }

    // Después de procesar todos los bloques de m.blocks,
    // si aún queda algún bloque en currentRowBlocks (un "left" al final de la lista),
    // lo renderizamos como "full".
    for _, pendingBlock := range currentRowBlocks {
        pendingBlock.SetWidth(m.width)
        isPendingBlockFocused := false
        if len(m.focusableBlocksOrder) > m.focusIndex && m.focusableBlocksOrder[m.focusIndex] == pendingBlock {
            isPendingBlockFocused = true
        }
        renderedStr, renderedHeight := m.renderStyledBlock(pendingBlock, isPendingBlockFocused)
        finalOutputLines = append(finalOutputLines, renderedStr)
        m.blockLayoutInfo = append(m.blockLayoutInfo, struct{ Block block.Block; StartLine int; Height int }{
            Block: pendingBlock, StartLine: currentRenderLine, Height: renderedHeight,
        })
        currentRenderLine += renderedHeight        
    }

    return strings.Join(finalOutputLines, "\n")
}

// Función auxiliar simple para max
func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

// updateFocusableOrder construye la lista de bloques en el orden en que deben recibir el foco.
// Se llama al inicio y cuando la configuración de bloques cambie (ej. añadir/quitar dinámicamente).
func (m *model) updateFocusableOrder() {
    m.focusableBlocksOrder = nil // Reset

    var currentRowBlocks []block.Block
    for _, b := range m.blocks {
        position := b.GetPosition()
        if position == "left" {
            currentRowBlocks = append(currentRowBlocks, b)
        } else if position == "right" {
            if len(currentRowBlocks) > 0 && currentRowBlocks[0].GetPosition() == "left" {
                currentRowBlocks = append(currentRowBlocks, b)
                m.focusableBlocksOrder = append(m.focusableBlocksOrder, currentRowBlocks...) // Añadir ambos
                currentRowBlocks = nil
            } else {
                // Right sin left, añadirlo como full
                m.focusableBlocksOrder = append(m.focusableBlocksOrder, b)
                currentRowBlocks = nil
            }
        } else { // "full"
            // Añadir cualquier pendiente de la fila anterior como full
            if len(currentRowBlocks) > 0 {
                m.focusableBlocksOrder = append(m.focusableBlocksOrder, currentRowBlocks...)
                currentRowBlocks = nil
            }
            m.focusableBlocksOrder = append(m.focusableBlocksOrder, b)
        }
    }
    // Añadir cualquier pendiente final
    if len(currentRowBlocks) > 0 {
        m.focusableBlocksOrder = append(m.focusableBlocksOrder, currentRowBlocks...)
    }
}

// renderStyledBlock es una función auxiliar que renderiza un bloque individual
// con su título, borde y estilo de foco.
func (m *model) renderStyledBlock(b block.Block, isFocused bool) (string, int) {
    blockView := b.View()
    blockName := b.Name()

    // Estilo para el borde del bloque
    borderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color("240")) // Color de borde por defecto

    // Si el bloque tiene el foco, cambia el color del borde
    if isFocused {
        borderStyle = borderStyle.BorderForeground(lipgloss.Color("12")) // Color de borde para el foco
    }

    // Obtener los colores del tema para el título desde el bloque
    blockThemeColors := b.GetThemeColors()
    titleStyle := lipgloss.NewStyle().
        Background(lipgloss.Color(blockThemeColors.TitleBackground)).
        Foreground(lipgloss.Color(blockThemeColors.TitleForeground)).
        PaddingLeft(1).PaddingRight(1).
        Bold(true)

    title := titleStyle.Render(blockName)

    // Ajusta el ancho del contenido DENTRO del borde y el título.
    // El ancho total que se le dio al bloque es b.GetSetWidth().
    // Restamos 2 por los bordes laterales.
    contentWidth := b.GetSetWidth() - 2
    if contentWidth < 0 { contentWidth = 0 }

    // Asegura que el contenido se ajusta al ancho disponible antes de renderizarlo.
    wrappedContent := lipgloss.NewStyle().Width(contentWidth).Render(blockView)

    // Unir el título y el contenido verticalmente.
    blockContentWithTitle := lipgloss.JoinVertical(lipgloss.Left, title, wrappedContent)

    // Aplicar el estilo de borde final
    finalRenderedBlock := borderStyle.Width(b.GetSetWidth()).Render(blockContentWithTitle)

    // Calcular la altura total de este bloque renderizado
    totalHeight := strings.Count(finalRenderedBlock, "\n") + 1

    return finalRenderedBlock, totalHeight
}

func (m *model) View() string {
    if m.currentView == "expanded" {
    	// en vista expandida solo viewport de ese bloque
        return m.viewport.View()
    }	

    // Solo re-renderizar el contenido del dashboard si es necesario
    if m.dashboardNeedsRender {
        // Reconstruye el orden de los bloques enfocables (esto debe ser eficiente)
        m.updateFocusableOrder() 
        // Genera el contenido completo del dashboard
        dashboardContentString := m.renderDashboardContent()
        m.viewport.SetContent(dashboardContentString)
        m.dashboardNeedsRender = false // Resetear el flag
        
        // ¡Importante! Resetear los flags de "contenido cambiado" en cada bloque
        for _, b := range m.blocks {
            b.ResetContentChangedFlag()
        }
    }

    return m.viewport.View()

}

// main.go (nuevo método en model)
func (m *model) adjustViewportToFocus() {
    if m.focusIndex < 0 || m.focusIndex >= len(m.focusableBlocksOrder) {
        return // Índice de foco inválido
    }

    focusedBlock := m.focusableBlocksOrder[m.focusIndex]
    
    for _, info := range m.blockLayoutInfo {
        if info.Block == focusedBlock {
            m.viewport.SetYOffset(info.StartLine)
            break
        }

    }

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
