// blocks/system_info/system_info.go
package system_info

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/shared/block"
    "github.com/gas/fancy-welcome/shared/messages" // Importa el nuevo paquete
)

type SystemInfoBlock struct {
	id    string
	style lipgloss.Style
	info  string
	updateInterval 	time.Duration
	nextRunTime 	time.Time
	position string
	width    int
    blockTheme *themes.Theme // Nuevo campo para almacenar el tema	
	renderedHeight       int  // Altura del bloque en su última renderización
    contentChangedSinceLastView bool // Nuevo flag
}

func New() block.Block {
	return &SystemInfoBlock{}
}

func (b *SystemInfoBlock) Name() string {
	return b.id
}

func (b *SystemInfoBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error {
	b.id = blockConfig["name"].(string)
	b.style = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Text))	

	var updateSecs float64 = 0
	// Se busca la clave "update_seconds".
	if val, ok := blockConfig["update_seconds"]; ok {
		// Se usa un type switch para manejar de forma segura int o float.
		switch v := val.(type) {
		case float64:
			updateSecs = v
		case int:
			updateSecs = float64(v)
		case int64:
			updateSecs = float64(v)
		}
	}
	// Si el valor es inválido (0, negativo) o no se encontró, se usa el global.
	if updateSecs <= 0 {
		updateSecs = globalConfig.GlobalUpdateSeconds
	}
	b.updateInterval = time.Duration(updateSecs) * time.Second

	// posición, y tema
	if pos, ok := blockConfig["position"].(string); ok {
		b.position = pos
	} else {
		b.position = "full" // Valor por defecto
	}
    b.blockTheme = theme // Almacenar el tema

	b.nextRunTime = time.Now()
	return nil
}

func (b *SystemInfoBlock) SetWidth(width int) {
	b.width = width
}

func (b *SystemInfoBlock) GetPosition() string {
	return b.position
}

func (b *SystemInfoBlock) GetSetWidth() int {
    return b.width
}

func (b *SystemInfoBlock) GetThemeColors() themes.ThemeColors {
    return b.blockTheme.Colors
}

func (b *SystemInfoBlock) Update() tea.Cmd {
	//2 opciones, cada una tiene sus problemas, usamos la más robusta
	//if time.Since(b.lastUpdateTime) < b.updateInterval { // el tiempo se gestiona en main.go
	//if time.Since(b.lastUpdateTime).Round(time.Second) < b.updateInterval { //redondea para evitar adelantos
	if time.Now().Before(b.nextRunTime) {
		return nil // No es hora de actualizar
	}

	return func() tea.Msg {
		// Helper to run a command and trim output
		getOutput := func(name string, arg ...string) string {
			out, err := exec.Command(name, arg...).Output()
			if err != nil {
				return "N/A"
			}
			return strings.TrimSpace(string(out))
		}

		hostname := getOutput("hostname")
		osInfo := getOutput("lsb_release", "-ds")
		kernel := getOutput("uname", "-r")

		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("Hostname: %s\n", hostname))
		builder.WriteString(fmt.Sprintf("OS:       %s\n", osInfo))
		builder.WriteString(fmt.Sprintf("Kernel:   %s\n", kernel))
		
		return messages.InfoMsg{BlockID: b.id, Info: builder.String()}
	}
}

func (b *SystemInfoBlock) View() string {
	var content string

	if b.info == "" {
		return b.style.Render("Loading system info...")
	}

	content = b.style.Render(b.info)
    //blockView := content // ... (el string que b.View() normalmente devuelve)
    //currentHeight := strings.Count(blockView, "\n") + 1 // Contar líneas, +1 para la última línea sin \n

	return content
}

func (b *SystemInfoBlock) RenderedHeight() int {
    // Esto es una ESTIMACIÓN. La altura final la determina el renderizador en main.go
    // cuando aplica word-wrap con el ancho final.
    // Por ahora, cuenta las líneas de la información base.
    if b.info == "" {
        return 1 // Altura para "Loading system info..."
    }
    // Si tu info se renderiza con newlines, strings.Count(b.info, "\n") + 1 es una estimación.
    // Si el renderizado real hace word-wrap, es más complejo y esto no será exacto.
    // La estrategia más robusta es que renderStyledBlock devuelva la altura y se cachee en el modelo.
    renderedContent := b.style.Render(b.info) // SystemInfoBlock no usa un renderer externo
    return strings.Count(renderedContent, "\n") + 1
}


// Nuevo: Se activa cuando los datos REALMENTE cambian.
func (b *SystemInfoBlock) HasContentChanged() bool {
    return b.contentChangedSinceLastView
}

// Nuevo: Se resetea después de que el dashboard lo ha procesado.
func (b *SystemInfoBlock) ResetContentChangedFlag() {
    b.contentChangedSinceLastView = false
}

// HandleMsg implements a specific handler for this block
func (b *SystemInfoBlock) HandleMsg(msg tea.Msg) {
	if m, ok := msg.(messages.InfoMsg); ok && m.BlockID == b.id {
		b.info = m.Info
		//b.lastUpdateTime = time.Now() // Actualiza el tiempo		
		// Calculamos la siguiente ejecución basándonos en la hora objetivo anterior.
		b.nextRunTime = b.nextRunTime.Add(b.updateInterval)

		//miramos si cambió la altura
        //if m.err == nil { // Solo si los datos se actualizaron correctamente
        //   b.contentChangedSinceLastView = true // Marcar que el contenido ha cambiado
        //}		
	}
}
