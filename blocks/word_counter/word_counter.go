package word_counter

import (
    "fmt"
    "strings"
    //"time"

    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss" // Importamos lipgloss para el estilo
    "github.com/gas/fancy-welcome/config"
    "github.com/gas/fancy-welcome/shared/block"
    "github.com/gas/fancy-welcome/themes"
)

// Struct más limpio, solo con los campos necesarios.
type WordCounterBlock struct {
    id          string
    style       lipgloss.Style
    listensTo   string // A qué bloque escucha
    countString string // Qué palabra o texto buscar
    count       int    // Su estado interno
    position    string
}

func New() block.Block { return &WordCounterBlock{} }

// --- MÉTODOS REQUERIDOS POR LA INTERFAZ ---

func (b *WordCounterBlock) Name() string {
    return b.id
}

func (b *WordCounterBlock) Position() string {
    return b.position
}

func (b *WordCounterBlock) RendererName() string {
    // No usa un renderer complejo, pero cumplimos el contrato.
    return "raw_text"
}

func (b *WordCounterBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error {
    b.id = blockConfig["name"].(string)
    b.listensTo, _ = blockConfig["listens_to"].(string)
    b.countString, _ = blockConfig["count_string"].(string)
    b.position, _ = blockConfig["position"].(string)
    b.count = 0
    
    // Aunque no lo usemos mucho, es bueno tener un estilo base.
    b.style = lipgloss.NewStyle().
        Background(lipgloss.Color(theme.Colors.Background)).
        Foreground(lipgloss.Color(theme.Colors.Text))
        
    return nil
}

func (b *WordCounterBlock) Update(p *tea.Program, msg tea.Msg) (block.Block, tea.Cmd) {
    switch m := msg.(type) {
    case block.TeeOutputMsg:
        // Tu lógica, que es perfecta, se mantiene.
        if m.SourceBlockID == b.listensTo {
            // Hacemos un type switch para ver qué tipo 
            // de datos nos han enviado. string o []string
            switch data := m.Output.(type) {
            case string:
                if text, ok := m.Output.(string); ok {
                    if strings.Contains(text, b.countString) {
                        b.count++
                    }
                }
            case []string:
                // Caso 2: Recibimos un lote de líneas (de un comando en streaming)
                for _, line := range data {
                    if strings.Contains(line, b.countString) {
                        b.count++
                    }
                }
            }
        }
    case block.StreamLineBatchMsg: // <-- Espera el mensaje de lote
        // ¿Es un mensaje de la fuente que me interesa?
        if m.BlockID() == b.listensTo {
            // Procesamos cada línea del lote
            for _, line := range m.Lines {
                if strings.Contains(line, b.countString) {
                    b.count++
                }
            }
        }
    }

    // Este bloque no necesita programar sus propios Ticks, ya que reacciona a otros.
    return b, nil
}

func (b *WordCounterBlock) View() string {
    // Aplicamos el estilo base al texto.
    viewString := fmt.Sprintf("'%s' Visto en '%s': %d veces", b.countString, b.listensTo, b.count)
    return b.style.Render(viewString)
}