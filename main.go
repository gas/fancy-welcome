// main.go
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

    // Asegúrate de que la ruta de importación coincida con tu módulo de Go
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/themes"
)

type model struct {
	message string
	style   lipgloss.Style
}

func initialModel(cfg *config.Config, theme *themes.Theme) model {
	// Extraer el mensaje del bloque 'hello' que definimos en el TOML
	helloBlockConfig, _ := cfg.Blocks["hello"].(map[string]interface{})
	message, _ := helloBlockConfig["message"].(string)

	// Crear un estilo usando los colores del tema cargado
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.Colors.Primary)).
		Background(lipgloss.Color(theme.Colors.Background)).
		Padding(2)

	return model{
		message: message,
		style:   style,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.style.Render(m.message)
}

func main() {
	// Cargar configuración
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error cargando la configuración: %v", err)
	}

	// Cargar tema
	theme, err := themes.LoadTheme(cfg.Theme.SelectedTheme)
	if err != nil {
		log.Fatalf("Error cargando el tema: %v", err)
	}

	// Iniciar la aplicación Bubble Tea
	p := tea.NewProgram(initialModel(cfg, theme))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error ejecutando el programa: %v\n", err)
		os.Exit(1)
	}
}
