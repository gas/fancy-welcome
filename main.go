// main.go
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gas/fancy-welcome/blocks/shell_command"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/shared/block"
	"github.com/gas/fancy-welcome/themes"
)

// blockFactory es un mapa que relaciona un 'type' de bloque con su función constructora.
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
	// Al iniciar, pedimos a cada bloque que se actualice.
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
	
	// Si el mensaje es de datos, lo pasamos al bloque correspondiente para que lo maneje.
    // Esta es una simplificación, una implementación más robusta necesitaría saber a qué bloque
    // pertenece el mensaje. Por ahora, se lo pasamos a todos.
	default:
		for _, b := range m.blocks {
			// Un truco para que los bloques puedan manejar sus propios mensajes
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

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error cargando config: %v", err)
	}

	theme, err := themes.LoadTheme(cfg.Theme.SelectedTheme)
	if err != nil {
		log.Fatalf("Error cargando tema: %v", err)
	}
	baseStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Text))

	var activeBlocks []block.Block
	for _, blockName := range cfg.General.EnabledBlocksOrder {
		blockConfig, ok := cfg.Blocks[blockName].(map[string]interface{})
		if !ok {
			log.Printf("Advertencia: config del bloque '%s' no encontrada o inválida.", blockName)
			continue
		}
		
		blockType, _ := blockConfig["type"].(string)
		if factory, ok := blockFactory[blockType]; ok {
			b := factory()
			blockConfig["name"] = blockName // Inyectamos el nombre en la config
			err := b.Init(blockConfig, baseStyle.Copy())
			if err != nil {
				log.Printf("Error inicializando bloque '%s': %v", blockName, err)
				continue
			}
			activeBlocks = append(activeBlocks, b)
		} else {
			log.Printf("Advertencia: tipo de bloque '%s' desconocido.", blockType)
		}
	}
	
	m := &model{blocks: activeBlocks}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error ejecutando el programa: %v\n", err)
		os.Exit(1)
	}
}
