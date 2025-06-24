// modes/welcome.go
package modes

import (
    "fmt"
    //"log"
    //"os"
    //"flag"
    //"path/filepath"

    //"github.com/charmbracelet/bubbles/textinput"    
    "github.com/charmbracelet/bubbles/viewport"
    "github.com/charmbracelet/bubbletea"
    //"github.com/charmbracelet/lipgloss"
    //"github.com/mattn/go-isatty"
    //"github.com/urfave/cli/v2"

    //"github.com/gas/fancy-welcome/config"
    //"github.com/gas/fancy-welcome/logging"
    "github.com/gas/fancy-welcome/shared/block"
    //"github.com/gas/fancy-welcome/shared/layout"
    //"github.com/gas/fancy-welcome/themes"
    //"github.com/gas/fancy-welcome/blocks/shell_command"
    //"github.com/gas/fancy-welcome/blocks/system_info"
    //"github.com/gas/fancy-welcome/blocks/word_counter"
    //"github.com/gas/fancy-welcome/blocks/filter"

    //"github.com/gas/fancy-welcome/modes" 
)

// WelcomeModel es el modelo de estado SOLO para el subcomando 'welcome'.
type WelcomeModel struct {
    blocks      []block.Block
    width       int
    height      int
    focusIndex  int
    viewport    viewport.Model
    // ...estilos de borde, etc.
    // ¡NOTA: No hay 'textInput' ni 'mode' string aquí!
}

// Init inicializa el estado y los comandos para el modo welcome.
func (m WelcomeModel) Init() tea.Cmd {
    // Tu lógica de Init para el dashboard.
    return func() tea.Msg { return block.TriggerUpdateMsg{} }
    //return block.TriggerUpdateMsg{}
}

// Update maneja los mensajes SOLO para el modo welcome.
func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Aquí copias la lógica de tu antiguo Update que NO estaba dentro del 'if m.mode == "creating_filter"'.
    // Manejarás el cambio de foco, el scroll del viewport, 'enter' para expandir, 'q' para salir, etc. 
    // ...
    return m, nil // Devuelve el modelo actualizado y los comandos
}

// View renderiza la UI del modo welcome.
func (m WelcomeModel) View() string {
    // Aquí copias la lógica de tu antiguo View que renderizaba el dashboard.
    // ...
    return m.viewport.View()
}

// RunWelcomeTUI lanza la aplicación interactiva para 'welcome'.
func RunWelcomeTUI(refreshTarget string) error {
    fmt.Println("TODO: Implementar modo TUI para 'welcome'")
    // Aquí irá la lógica de Bubble Tea para 'welcome'
    
    // Creas el modelo específico.
    initialModel := WelcomeModel{
        // ... inicializas los campos ...
    }

    p := tea.NewProgram(initialModel, tea.WithAltScreen())
    _, err := p.Run()
    return err
}

// RunWelcomeTTY ejecuta la lógica de volcado en texto plano para 'welcome'.
func RunWelcomeTTY(refreshTarget string) error {
    fmt.Println("TODO: Implementar modo TTY para 'welcome'")
    // Aquí irá la lógica de ejecución síncrona y volcado a consola
    return nil
}