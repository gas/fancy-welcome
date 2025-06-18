package filter

import (
	"fmt"
	"github.com/charmbracelet/bubbletea"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/shared/block"
	"github.com/gas/fancy-welcome/themes"
)

// Este es un esqueleto básico.
type FilterBlock struct {
	id        string
	listensTo string
	filter    string
}

func New() block.Block { return &FilterBlock{} }

func (b *FilterBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error {
	b.id = blockConfig["name"].(string)
	b.listensTo, _ = blockConfig["listens_to"].(string)
	b.filter, _ = blockConfig["filter"].(string)
	return nil
}

// Por ahora, el Update no hace nada más que escuchar.
func (b *FilterBlock) Update(p *tea.Program, msg tea.Msg) (block.Block, tea.Cmd) {
	// Aquí iría la lógica para procesar TeeOutputMsg
	return b, nil
}

func (b *FilterBlock) View() string {
	return fmt.Sprintf("Filtro para '%s' buscando '%s'", b.listensTo, b.filter)
}

// Métodos para cumplir la interfaz
func (b *FilterBlock) Name() string     { return b.id }
func (b *FilterBlock) Position() string { return "full-width" } // O leer de config
func (b *FilterBlock) RendererName() string { return "raw_text" }