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
	"github.com/gas/fancy-welcome/shared/block"
)

type SystemInfoBlock struct {
	id    string
	style lipgloss.Style
	info  string
	updateInterval time.Duration
	lastUpdateTime time.Time	
}

func New() block.Block {
	return &SystemInfoBlock{}
}

func (b *SystemInfoBlock) Name() string {
	return b.id
}

func (b *SystemInfoBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, style lipgloss.Style) error {
	b.id = blockConfig["name"].(string)
	b.style = style
	
	updateSecs, ok := blockConfig["update_seconds"].(float64)
	if !ok || updateSecs <= 0 {
		updateSecs = globalConfig.GlobalUpdateSeconds
	}
	b.updateInterval = time.Duration(updateSecs) * time.Second

	return nil
}

func (b *SystemInfoBlock) Update() tea.Cmd {
	if time.Since(b.lastUpdateTime) < b.updateInterval {
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
		b.lastUpdateTime = time.Now() // Actualiza el tiempo		
	}
}