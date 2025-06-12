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
)

type SystemInfoBlock struct {
	id    string
	style lipgloss.Style
	info  string
	updateInterval 	time.Duration
	nextRunTime 	time.Time	
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

	b.nextRunTime = time.Now()
	return nil
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
		
		return infoMsg{blockID: b.id, info: builder.String()}
	}
}

func (b *SystemInfoBlock) View() string {
	if b.info == "" {
		return b.style.Render("Loading system info...")
	}
	return b.style.Render(b.info)
}

// Message for when info is fetched
type infoMsg struct {
	blockID string
	info    string
}

// HandleMsg implements a specific handler for this block
func (b *SystemInfoBlock) HandleMsg(msg tea.Msg) {
	if m, ok := msg.(infoMsg); ok && m.blockID == b.id {
		b.info = m.info
		//b.lastUpdateTime = time.Now() // Actualiza el tiempo		
		// Calculamos la siguiente ejecución basándonos en la hora objetivo anterior.
		b.nextRunTime = b.nextRunTime.Add(b.updateInterval)
	}
}