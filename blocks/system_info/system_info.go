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
	"github.com/gas/fancy-welcome/logging" // paquete de logging
	"github.com/gas/fancy-welcome/shared/block"
)

type SystemInfoBlock struct {
	id    			string
	style 			lipgloss.Style
	info  			string
	updateInterval 	time.Duration
    position     	string
	blockConfig    map[string]interface{}
	isLoading      bool
}

// Message for when info is fetched
type infoMsg struct {
	blockID string
	info    string
	err     error
}

func (m infoMsg) BlockID() string { return m.blockID } // <-- AÑADE ESTE MÉTODO

func (b *SystemInfoBlock) Position() string {
	if val, ok := b.blockConfig["position"].(string); ok {
		b.position = val
	}
    return b.position
}

func New() block.Block {
	return &SystemInfoBlock{}
}

func (b *SystemInfoBlock) Name() string {
	return b.id
}


func (b *SystemInfoBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error {
    logging.Log.Printf("SI Init: [%s] received msg: %T", b.id)
	b.blockConfig = blockConfig
	b.id = blockConfig["name"].(string)
    b.position, _ = blockConfig["position"].(string)
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

	return nil
}

func (b *SystemInfoBlock) Update(msg tea.Msg) (block.Block, tea.Cmd) {
	switch m := msg.(type) {
	// Este case ahora maneja DOS tipos de trigger:
	// 1. El TriggerUpdateMsg general del arranque.
	// 2. Un BlockTickMsg que sea para este bloque en específico.
	case block.TriggerUpdateMsg, block.BlockTickMsg:
		// Para BlockTickMsg, nos aseguramos de que es para nosotros.
		if tick, ok := m.(block.BlockTickMsg); ok {
			if tick.TargetBlockID != b.id {
				return b, nil // No es para mí, lo ignoro.
			}
		}

		// Si llegamos aquí, es nuestro turno de actualizar.
		if b.isLoading {
			return b, nil
		}

		b.isLoading = true
		// --- CORRECCIÓN DEL ERROR 1 ---
		// Devolvemos el bloque 'b' y un comando que genera el 'infoMsg'.
		return b, func() tea.Msg {
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

	case infoMsg:
		if m.blockID == b.id {
			b.isLoading = false
			b.info = m.info
			// --- CORRECCIÓN DEL ERROR 2 ---
			// Pasamos el ID del bloque y el intervalo.
			return b, block.ScheduleNextTick(b.id, b.updateInterval)
		}
	}

	return b, nil
}

/*
func (b *SystemInfoBlock) OLD_Update(msg tea.Msg) (block.Block, tea.Cmd)  {
    logging.Log.Printf("SI Update: [%s] received msg: %T", b.id, msg)

	switch msg := msg.(type) {
	// Mensaje para iniciar la actualización
	case block.TriggerUpdateMsg:
		if b.isLoading {
			return b, nil
		}
		b.isLoading = true
		// Devolvemos el bloque y el comando para cargar datos
		return b, func() tea.Msg {
			getOutput := func(name string, arg ...string) string {
				out, err := exec.Command(name, arg...).Output()
				if err != nil { return "N/A" }
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

	// Mensaje con el resultado de la carga
	case infoMsg:
		if msg.blockID == b.id {
			b.isLoading = false
			b.info = msg.info
			return b, block.ScheduleNextUpdate(b.updateInterval)
		}
	}
	return b, nil
}
*/

func (b *SystemInfoBlock) View() string {
	if b.info == "" && b.isLoading {
		return b.style.Render("Loading system info...")
	}

	if b.info == "" {
		return b.style.Render("...")
	}
	return b.style.Render(b.info)
}
