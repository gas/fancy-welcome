// modes/filter.go
package modes

import (
    "fmt"
    "log"
    "os"
    //"flag"
    //"path/filepath"
    //"github.com/mattn/go-isatty"
    //"github.com/urfave/cli/v2"

    // --- IMPORTS PARA EL MODELO DE BUBBLE TEA ---
    "github.com/charmbracelet/bubbles/textinput"    
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"


    // --- IMPORTS PARA LA LÓGICA DE LA APLICACIÓN ---
    "github.com/gas/fancy-welcome/config"
    "github.com/gas/fancy-welcome/shared"
    "github.com/gas/fancy-welcome/shared/block"
    "github.com/gas/fancy-welcome/themes"
    "github.com/gas/fancy-welcome/logging"

    //"github.com/gas/fancy-welcome/blocks/shell_command"
    //"github.com/gas/fancy-welcome/blocks/system_info"
    //"github.com/gas/fancy-welcome/blocks/word_counter"
    //"github.com/gas/fancy-welcome/blocks/filter"

    //"github.com/gas/fancy-welcome/modes" 
)

// FilterModel es el modelo de estado SOLO para el subcomando 'filter'.
type FilterModel struct {
    blocks           []block.Block
    viewport         viewport.Model // <-- Un solo viewport para todo
    focusIndex       int
    width, height    int
    isCreatingFilter bool
    textInput        textinput.Model
    expandedBlock    block.Block 

    // --- CAMPOS DE CONFIGURACIÓN Y ESTILOS ---
    blockFactory      map[string]func() block.Block
    theme             *themes.Theme
    globalConfig      config.GeneralConfig
    normalBorderStyle lipgloss.Style
    focusBorderStyle  lipgloss.Style
}

// NewFilterModel: El constructor se asegura de que el modelo se cree con todo lo necesario.
// func NewFilterModel(blocks []block.Block, factory map[string]func() block.Block, theme *themes.Theme, cfg config.GeneralConfig) FilterModel {
func NewFilterModel(setupResult *shared.SetupResult) FilterModel {
    ti := textinput.New()
    ti.Placeholder = "Término a filtrar (ej: -i -v ERROR)"
    // Hemos creado el textinput. No lo enfocamos hasta que el usuario pulse 'a'

    //El viewport se crea aquí una sola vez. Antes en RunTuiMode,
    vp := viewport.New(100, 20) // El tamaño se ajustará con el primer WindowSizeMsg
    vp.Style = lipgloss.NewStyle().
        Background(lipgloss.Color(setupResult.Theme.Colors.Background)).
        Foreground(lipgloss.Color(setupResult.Theme.Colors.Text))

    // Los estilos de borde también se definen aquí.
    normalBorderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color(setupResult.Theme.Colors.Border))

    focusBorderStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(lipgloss.Color(setupResult.Theme.Colors.Primary))

    return FilterModel{
        blocks:            setupResult.ActiveBlocks,
        isCreatingFilter:  false,
        textInput:         ti,
        blockFactory:      setupResult.BlockFactory,
        theme:             setupResult.Theme,
        globalConfig:      setupResult.Config.General,
        viewport:          vp,
        normalBorderStyle: normalBorderStyle,
        focusBorderStyle:  focusBorderStyle,
    }
}



// Init inicializa el modo filtro.
func (m FilterModel) Init() tea.Cmd {
    // Puede que queramos un TriggerUpdateMsg inicial para los bloques
    return func() tea.Msg { return block.TriggerUpdateMsg{} }
}

// Update msgs para el modo filtro.
func (m FilterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    logging.Log.Printf("FilterModel --- UPDATE RECEIVED MSG TYPE: %T ---", msg)

    var cmd tea.Cmd
    var cmds []tea.Cmd

    // p := m.program // Mejor es no tenerlo en el modelo.

    // === MÁQUINA DE ESTADOS ===

    // --- ESTADO 1: VISTA EXPANDIDA (Máxima prioridad) ---
    if m.expandedBlock != nil {
        switch msg := msg.(type) {
        case tea.KeyMsg:
            switch msg.String() {
            // Volver al dashboard
            case "q", "esc", "enter":
                m.expandedBlock = nil // <-- La clave: volvemos al estado dashboard
                // Restauramos el contenido del dashboard en el viewport
                m.viewport.SetContent(m.renderDashboardView()) // Usamos una función helper
                return m, nil

            // --- Guardar a archivo ---
            case "s":
                // Obtenemos el contenido expandido del bloque actual
                content := m.expandedBlock.View() // Por defecto
                if expander, ok := m.expandedBlock.(block.Expander); ok {
                    content = expander.ExpandedView() // Usamos la vista expandida si existe 
                }
                
                // Creamos un nombre de archivo único
                fileName := fmt.Sprintf("%s_expanded_view.txt", m.expandedBlock.Name())
                
                // Escribimos a disco
                err := os.WriteFile(fileName, []byte(content), 0644)
                if err != nil {
                    // En una app real, enviarías un tea.Msg de error para mostrarlo en la UI
                    log.Printf("Error guardando archivo: %v", err)
                }
                // Podrías enviar un mensaje de confirmación también
                return m, nil
            }            
        }
        
        // Si no es una tecla de control, pasamos el mensaje al viewport para el scroll
        m.viewport, cmd = m.viewport.Update(msg)
        cmds = append(cmds, cmd)
        return m, tea.Batch(cmds...) 
        // return m, cmd
    }


    // ESTADO 2: CREANDO UN FILTRO
    if m.isCreatingFilter {
        switch msg := msg.(type) {
        case tea.KeyMsg:
            switch msg.Type {
            // Caso 1: El usuario pulsa Enter -> Creamos el bloque
            case tea.KeyEnter: // o msg.String()
                filterQuery := m.textInput.Value() // 
                parentBlock := m.blocks[m.focusIndex] // 
                parentName := parentBlock.Name()

                // 3. Creamos la configuración para el nuevo bloque.
                newBlockName := fmt.Sprintf("%s_filter_%d", parentName, len(m.blocks)+1) // 
                newBlockConfig := map[string]interface{}{
                    "name":       newBlockName,
                    "type":       "Filter",
                    "listens_to": parentName,
                    "filter":     filterQuery,
                }

                // 4. Creamos e inicializamos el nuevo bloque.
                factory := m.blockFactory[newBlockConfig["type"].(string)] // 
                newBlock := factory()
                newBlock.Init(newBlockConfig, m.globalConfig, m.theme)

                // 5. Lo insertamos en el slice justo después de su padre.
                insertionIndex := m.focusIndex + 1 // 
                m.blocks = append(m.blocks[:insertionIndex], append([]block.Block{newBlock}, m.blocks[insertionIndex:]...)...) // 

                // 6. Volvemos al modo normal y reseteamos el input.
                m.isCreatingFilter = false
                m.textInput.Reset() 
                m.textInput.Blur() // ??
                return m, nil

            // Caso 2: El usuario pulsa Escape -> Cancelamos
            case tea.KeyEsc:
                m.isCreatingFilter = false // 
                m.textInput.Reset()      // 
                m.textInput.Blur() // ??
                return m, nil
            }
        }

        // Si no es Enter o Esc, pasamos el mensaje al textInput para que se actualice
        m.textInput, cmd = m.textInput.Update(msg) // 
        return m, cmd
    }


    // ESTADO 3: NAVEGACIÓN NORMAL
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height
        return m, nil 

    case tea.KeyMsg:
        switch msg.String() {
        // El usuario pulsa 'a' para AÑADIR un filtro
        case "a":
            m.isCreatingFilter = true  // Cambiamos al modo de creación
            m.textInput.Focus()        // 
            return m, textinput.Blink // 

        case "q", "ctrl+c":
            return m, tea.Quit

        case "enter":
            focusedBlock := m.blocks[m.focusIndex]
            m.expandedBlock = focusedBlock
            content := focusedBlock.View()

            // El bloque enfocado tiene una vista expandida?
            if expander, ok := focusedBlock.(block.Expander); ok {
                content = expander.ExpandedView()
            } 
            m.viewport.SetContent(content)
            m.viewport.GotoTop()
            return m, nil

        case "tab":
            m.focusIndex = (m.focusIndex + 1) % len(m.blocks)
            m.viewport.SetContent(m.renderDashboardView()) // Refrescamos para que se vea el nuevo borde
            return m, nil

        case "up", "k", "down", "j", "pgup", "pgdown":
            m.viewport, cmd = m.viewport.Update(msg) // Pasa el mensaje al viewport principal
            cmds = append(cmds, cmd)
        }
    
    
    default:
        // --- BUCLE DE DELEGACIÓN ---
        // ¿Es un mensaje dirigido a un bloque específico?
        if targetMsg, ok := msg.(block.TargetedMsg); ok {
            // Sí, es dirigido. Buscamos el bloque objetivo.
            targetID := targetMsg.BlockID()
            for i, b := range m.blocks {
                if b.Name() == targetID {
                    updatedBlock, blockCmd := b.Update(msg)
                    m.blocks[i] = updatedBlock 
                    cmds = append(cmds, blockCmd)
                    break // out of tha loop
                }
            }
        } else {
            // No, es un mensaje general (como TriggerUpdateMsg o spinner.TickMsg).
            // Lo enviamos a todos los bloques.
            for i, b := range m.blocks {
                updatedBlock, blockCmd := b.Update(msg) //<-- STREAM
                m.blocks[i] = updatedBlock
                cmds = append(cmds, blockCmd)
            }           
        }
    }

    return m, tea.Batch(cmds...)
}

// View renderiza la UI del modo filtro.
func (m FilterModel) View() string {
    // Si estamos expandidos, el viewport ya tiene el contenido correcto.
    if m.expandedBlock != nil {
        return m.viewport.View()
    }

    // Obtenemos el contenido del dashboard llamando a la función compartida.
    dashboardContent := shared.RenderDashboard(
        m.width, 
        m.blocks, 
        m.focusIndex, 
        m.normalBorderStyle, 
        m.focusBorderStyle,
    )

    m.viewport.SetContent(dashboardContent)
    mainView := m.viewport.View()

    // podemos usar el input para más acciones si lo necesitamos...
    // tb para el modo inline interactivo como ctrl+R en fzf...

    // Y si estamos creando un filtro, le añadimos el input.
    if m.isCreatingFilter {
        //dashboardContent := m.renderDashboardView()
        inputView := fmt.Sprintf(
            "\n\nAñadir filtro para '%s' (Enter para aceptar, Esc para cancelar):\n%s",
            m.blocks[m.focusIndex].Name(),
            m.textInput.View(),
        )
        // Unimos el dashboard (sin el viewport) con el input.
        return lipgloss.JoinVertical(lipgloss.Left, dashboardContent, inputView)
        // return dashboardView + inputView
    }

    // Si no, simplemente mostramos el dashboard a través del viewport.
    //m.viewport.SetContent(dashboardContent)
    //return m.viewport.View()
    return mainView
}

// --- HELPER: renderDashboardView ---
// para componer los bloques en un solo string.
func (m FilterModel) renderDashboardView() string {

    if m.expandedBlock != nil {
        // Si estamos en vista expandida, mostramos ESE viewport
        return m.viewport.View()
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
    m.viewport.SetContent(fullLayout)

    // 3. Devolvemos la vista del viewport, que se encargará del scroll.
    return m.viewport.View()
}



// --- RUNNERS: implementamos el TUI runner correctamente ---

// RunFilterTUI lanza la aplicación interactiva para 'filter'.
func RunFilterTUI() error {
    fmt.Println("Modo TUI para 'fancy filter'")

    setupResult, err := shared.Setup("") // Llama a la función centralizada
    if err != nil { 
        return fmt.Errorf("error al inicializar la configuración: %w", err)
    }

    // Creamos el modelo usando el constructor.    
    initialModel := NewFilterModel(setupResult) // Pasa el resultado al constructor
    p := tea.NewProgram(initialModel, tea.WithAltScreen(), tea.WithMouseAllMotion()) //?? util + o - que withMouseCellMotion?
 
    // Antes de ejecutar el programa, iteramos sobre los bloques iniciales del modelo.
    for _, b := range initialModel.blocks {
        // Hacemos una aserción de tipo: "¿Este bloque 'b' implementa la interfaz 'Streamer'?"
        if streamer, ok := b.(block.Streamer); ok {
            // Si la respuesta es "sí" (ok == true), entonces le pasamos el puntero.
            streamer.SetProgram(p)
        }
    }

    if _, err := p.Run(); err != nil {
        // Usamos log.Printf para que no cierre la aplicación con Fatalf y se vea el error TUI.
        log.Printf("Error al ejecutar el programa TUI: %v", err)
        return err
    }

    return nil
}

// RunFilterTTY ejecuta la lógica de volcado para 'filter'.
func RunFilterTTY() error {
    fmt.Println("TODO: Implementar modo TTY para 'filter'")
    return nil
}