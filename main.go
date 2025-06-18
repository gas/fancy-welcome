// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	//"strings"
	"path/filepath"
	//"time"

	//"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/bubbles/textinput"	
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"github.com/gas/fancy-welcome/blocks/shell_command"
	"github.com/gas/fancy-welcome/blocks/system_info"
	"github.com/gas/fancy-welcome/blocks/word_counter"
    "github.com/gas/fancy-welcome/blocks/filter"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/shared/block"
	"github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/logging"
)

// --- El código del modo interactivo (Bubble Tea) permanece igual ---

var blockFactory = map[string]func() block.Block{
	"ShellCommand": shell_command.New,
	"SystemInfo":   system_info.New,
	"WordCounter":   word_counter.New,
	"Filter":   filter.New,
}


type model struct {
    program 	*tea.Program // <-- STREAM
	blocks 		[]block.Block
	width  		int
	height 		int
    currentView string // "dashboard" o "viewport"
    viewport    viewport.Model
    focusIndex  		int    // Índice del bloque que tiene el foco	
	dashboardVP 		viewport.Model // Para la vista principal
	expandedVP  		viewport.Model  // Para la vista de un solo bloque
	normalBorderStyle 	lipgloss.Style
	focusBorderStyle  	lipgloss.Style
	mode      	string  	// "dashboard", "creating_filter", etc.
	textInput 	textinput.Model  
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

	// Si estamos en modo de entrada de texto, solo nos interesa el textInput.
	if m.mode == "creating_filter" {
		var cmd tea.Cmd
		
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEnter:
				// 1. Obtenemos el texto introducido.
				filterQuery := m.textInput.Value()
				
				// 2. Obtenemos el bloque "padre" que tenía el foco.
				parentBlock := m.blocks[m.focusIndex]
				parentName := parentBlock.Name()

				// 3. Creamos la configuración para el nuevo bloque.
				newBlockName := fmt.Sprintf("%s_filter_%d", parentName, len(m.blocks)+1) // Usamos len+1 para un ID único
				newBlockConfig := map[string]interface{}{
					"name":       newBlockName,
					"type":       "Filter",
					"listens_to": parentName,
					"filter":     filterQuery, // Pasamos la query del usuario
				}

				// 4. Creamos e inicializamos el nuevo bloque.
				factory := blockFactory[newBlockConfig["type"].(string)]
				newBlock := factory()
				newBlock.Init(newBlockConfig, config.GeneralConfig{}, &themes.Theme{})

				// 5. Lo insertamos en el slice justo después de su padre.
				insertionIndex := m.focusIndex + 1
				m.blocks = append(m.blocks[:insertionIndex], append([]block.Block{newBlock}, m.blocks[insertionIndex:]...)...)
	
				// 6. Volvemos al modo normal y reseteamos el input.
				m.mode = "dashboard"
				m.textInput.Reset()
				return m, nil

			case tea.KeyEsc:
				// El usuario ha cancelado.
				m.mode = "dashboard"
				m.textInput.Reset()
				return m, nil
			}
		}

		// Pasamos el mensaje al textInput para que se actualice (el usuario está escribiendo).
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}

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

			// --- LÓGICA INTELIGENTE ---
			// ¿El bloque enfocado tiene una vista expandida?
			if expander, ok := focusedBlock.(block.Expander); ok {
				// Sí, la tiene. Usamos su vista expandida.
				m.expandedVP.SetContent(expander.ExpandedView())
			} else {
				// No, no la tiene. Usamos su vista normal como fallback.
				m.expandedVP.SetContent(focusedBlock.View())
			}

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
		case "a":
			m.mode = "creating_filter"
			m.textInput.Focus() // Activamos el cursor en el campo de texto
			return m, textinput.Blink // Devuelve el comando para que el cursor parpadee
/*
			// 1. Identificar el bloque "padre" usando el foco actual.
			//    Añadimos un chequeo de seguridad por si acaso.
			if m.focusIndex < 0 || m.focusIndex >= len(m.blocks) {
				return m, nil // No hacer nada si el foco es inválido
			}
			parentBlock := m.blocks[m.focusIndex]
			parentName := parentBlock.Name()

			// 1. Creamos la configuración para el nuevo bloque en memoria.
			newBlockName := fmt.Sprintf("%s_filter_%d", parentName, len(m.blocks))
			newBlockConfig := map[string]interface{}{
				"name":       newBlockName,
				"type":       "Filter",
				"listens_to": parentName,
				"filter":     "TICK", // Un filtro de ejemplo
			}

			// 2. Usamos nuestra factory para crear la instancia.
			if factory, ok := blockFactory[newBlockConfig["type"].(string)]; ok {
				newBlock := factory()
				// 3. Inicializamos el bloque.
				// Pasamos un cfg y theme vacíos porque este bloque no los necesita mucho.
				newBlock.Init(newBlockConfig, config.GeneralConfig{}, &themes.Theme{})

				// 4. Insertamos el nuevo bloque en la posición correcta.
				//    Esta es la forma idiomática de insertar en un slice en Go.
				insertionIndex := m.focusIndex + 1
				m.blocks = append(m.blocks[:insertionIndex], append([]block.Block{newBlock}, m.blocks[insertionIndex:]...)...)
				
				// Opcional: movemos el foco directamente al nuevo bloque creado.
				m.focusIndex = insertionIndex
			}
			
			return m, nil // Forzamos un redibujado*/
        }
    

    default:
	    // --- BUCLE DE DELEGACIÓN ---
		// ¿Es un mensaje dirigido a un bloque específico?
		if targetMsg, ok := msg.(block.TargetedMsg); ok {
			// Sí, es dirigido. Buscamos el bloque objetivo.
			targetID := targetMsg.BlockID()
		    for i, b := range m.blocks {
				if b.Name() == targetID {
					updatedBlock, cmd := b.Update(m.program,msg)
		        	m.blocks[i] = updatedBlock 
		        	cmds = append(cmds, cmd)
		        	break // out of tha loop
		        }
		    }
	    } else {
			// No, es un mensaje general (como TriggerUpdateMsg o spinner.TickMsg).
			// Lo enviamos a todos los bloques.
			for i, b := range m.blocks {
				updatedBlock, cmd := b.Update(m.program, msg) //<-- STREAM
				m.blocks[i] = updatedBlock
				cmds = append(cmds, cmd)
			}	    	
	    }
	}

	return m, tea.Batch(cmds...)
}


func (m *model) View() string {
	// Primero, obtenemos la vista del dashboard como siempre.
	dashboardView := m.renderDashboard() // Nueva función helper

	// Si estamos en modo de creación de filtro...
	if m.mode == "creating_filter" {
		// ...dibujamos el dashboard y, debajo, el campo de texto.
		return lipgloss.JoinVertical(lipgloss.Left,
			dashboardView,
			"\n\nAñadir filtro (Enter para aceptar, Esc para cancelar):",
			m.textInput.View(),
		)
	}

	// Si no, simplemente devolvemos la vista del dashboard.
	return dashboardView
}

func (m *model) renderDashboard() string {

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
    	var renderedBlock string // Declaramos la variable que contendrá el resultado final
    	//logging.Log.Printf("FOCUS CHECK: focusIndex=%d, blockIndex=%d, blockName=%s", m.focusIndex, i, b.Name())

/*    	// ETIQUETA 'a: Add Filter' en el borde inferior 
		var finalBlockView string // Un string para el contenido final del bloque (vista + tag)

		if i == m.focusIndex {
		    // Creamos el tag solo para el bloque con foco
		    // lipgloss.Place es perfecto para alinear el texto
		    tag := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFF")).Background(lipgloss.Color("#FF5F87")).Render(" a: add filter ")
		    
		    // Obtenemos el ancho del bloque para alinear el tag a la derecha
		    blockWidth := (m.width / 2) - 4 // Asumiendo columna (ajustar si es full-width)
		    alignedTag := lipgloss.PlaceHorizontal(blockWidth, lipgloss.Right, tag)

		    // Unimos la vista del bloque y el tag verticalmente
		    finalBlockView = lipgloss.JoinVertical(lipgloss.Left, blockView, alignedTag)
		} else {
		    // Si el bloque no tiene foco, su vista final es solo su contenido
		    finalBlockView = blockView
		}
		// Ahora, el borde se aplica al conjunto de 'finalBlockView'
		var borderStyle lipgloss.Style
		// ... (la lógica if/else para seleccionar el estilo del borde se mantiene igual)
		renderedBlock := borderStyle.Width(blockWidth).Render(finalBlockView)		
*/

	    // --- LÓGICA DE RENDERIZADO CONDICIONAL ---
	    if b.RendererName() == "preformatted_text" {
	        // Si es pre-formateado, usamos la vista cruda, sin bordes ni estilos.
	        renderedBlock = blockView
	    	logging.Log.Printf("PREFORMATTED: blockIndex=%d, blockName=%s", i, b.Name())

	    } else {
	        // Para todos los demás, aplicamos el borde y el estilo de foco.
	        var borderStyle lipgloss.Style
	        if i == m.focusIndex {
	            borderStyle = m.focusBorderStyle
	        } else {
	            borderStyle = m.normalBorderStyle
	        }

	        position := b.Position() 
	        if position == "left" || position == "right" {
	            blockWidth := (m.width / 2) - 4 
	            renderedBlock = borderStyle.Width(blockWidth).Render(blockView)
	        } else {
	            blockWidth := m.width - 2
	            renderedBlock = borderStyle.Width(blockWidth).Render(blockView)
	        }
	    }

	    // El resto de la lógica para añadir a las columnas se mantiene igual,
	    // pero ahora usa la variable 'renderedBlock'.
	    position := b.Position()
	    if position == "left" || position == "right" {
	         if position == "left" {
	            leftColumnViews = append(leftColumnViews, renderedBlock)
	        } else {
	            rightColumnViews = append(rightColumnViews, renderedBlock)
	        }
	    } else {
	         processPendingColumns()
	         finalLayout = append(finalLayout, renderedBlock)
	    }

    }

    // 3. Procesa cualquier columna que haya quedado al final del bucle.
    processPendingColumns()

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

	// Creamos la instancia del textInput
	ti := textinput.New()
	ti.Placeholder = "Término a filtrar (ej: -i -v ERROR)"
	ti.Focus() // Lo activamos por defecto, aunque solo lo mostraremos cuando sea necesario

	m := &model{
		blocks:      activeBlocks,
		currentView: "dashboard",
		dashboardVP: dashVP,
		expandedVP:  expVP,
		focusIndex:  0, // Inicia el foco en el primer bloque
		normalBorderStyle: normalBorderStyle,
		focusBorderStyle:  focusBorderStyle,
		mode:      "dashboard", // El modo inicial es el dashboard
		textInput: ti,		
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p // <-- STREAM
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
	    if isStreaming, ok := blockConfig["streaming"].(bool); ok && isStreaming {
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
		updatedBlock, cmd := b.Update(nil, block.TriggerUpdateMsg{})

		// Paso 3.2: Ejecutar el comando si es necesario.
		if cmd != nil {
			// Esta llamada es bloqueante: espera a que el comando termine y devuelve el mensaje.
			msg := cmd()
			
			// Paso 3.3: Finalizar la actualización.
			updatedBlock, _ = updatedBlock.Update(nil, msg)
		}

		// Paso 4: Imprimir la vista.
		// Ahora que el bloque está completamente actualizado (desde caché o ejecución),
		// imprimimos su vista directamente a la consola.
		fmt.Println(updatedBlock.View())
		fmt.Println("---") // Añadimos un separador simple
	}
}
