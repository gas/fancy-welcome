// blocks/shell_command/shell_command.go
package shell_command

import (
	"encoding/json"	
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/spinner"

	"github.com/gas/fancy-welcome/blocks/shell_command/parsers"
	"github.com/gas/fancy-welcome/blocks/shell_command/renderers"
	"github.com/gas/fancy-welcome/shared/block"
)

var registeredParsers = make(map[string]parsers.Parser)
var registeredRenderers = make(map[string]renderers.Renderer)

func init() {
	registeredParsers["single_line"] = &parsers.SingleLineParser{}
	registeredParsers["multi_line"] = &parsers.MultiLineParser{}
	
	registeredRenderers["raw_text"] = &renderers.RawTextRenderer{}
	registeredRenderers["cowsay"] = &renderers.CowsayRenderer{}
}

// Nuevo struct para guardar en el archivo de caché
type cacheEntry struct {
	Timestamp  time.Time   `json:"timestamp"`
	ParsedData interface{} `json:"parsed_data"`
}

// 1: Añadido el campo 'id' al struct del bloque.
type ShellCommandBlock struct {
	id           string // ID único del bloque
	style        lipgloss.Style
	command      string
	args         []string
	parser       parsers.Parser
	renderer     renderers.Renderer
	parsedData   interface{}
	currentError error
    //lastUpdated   time.Time
    cacheDuration time.Duration
    isLoading bool
    spinner   spinner.Model
}

// CORRECCIÓN 2: Añadir 'blockID' al mensaje para saber a quién pertenece.
type dataMsg struct {
	blockID string
	data    interface{}
	err     error
}

func New() block.Block {
	return &ShellCommandBlock{}
}

func (b *ShellCommandBlock) Name() string {
    // Usamos el id como el nombre, ya que es único.
	return b.id
}

func (b *ShellCommandBlock) Spinner() *spinner.Model { return &b.spinner }

func (b *ShellCommandBlock) SpinnerCmd() tea.Cmd { return b.spinner.Tick }

func (b *ShellCommandBlock) Init(config map[string]interface{}, style lipgloss.Style) error {
	// CORRECCIÓN 3: Guardamos el nombre único del bloque como su ID.
	b.id = config["name"].(string) 
	b.style = style

	cmdRaw, _ := config["command"].(string)
	if cmdRaw == "" {
		return fmt.Errorf("el campo 'command' es obligatorio para el bloque '%s'", b.id)
	}
	parts := strings.Fields(cmdRaw)
	b.command = parts[0]
	if len(parts) > 1 {
		b.args = parts[1:]
	}

	parserName, _ := config["parser"].(string)
	p, ok := registeredParsers[parserName]
	if !ok {
		return fmt.Errorf("parser '%s' no encontrado para el bloque '%s'", parserName, b.id)
	}
	b.parser = p

	rendererName, _ := config["renderer"].(string)
	r, ok := registeredRenderers[rendererName]
	if !ok {
		return fmt.Errorf("renderer '%s' no encontrado para el bloque '%s'", rendererName, b.id)
	}
	b.renderer = r

    // Leer la duración de la caché en segundos
    if cacheSecs, ok := config["cache_duration_seconds"].(int64); ok {
        b.cacheDuration = time.Duration(cacheSecs) * time.Second
    } else {
        // Valor por defecto si no se especifica (ej. 10 minutos)
        b.cacheDuration = 10 * time.Minute
    }

    b.spinner = spinner.New()
    // Podemos estilizar el spinner usando los colores del tema
    b.spinner.Style = lipgloss.NewStyle().Foreground(style.GetForeground())

	return nil
}

func (b *ShellCommandBlock) Update() tea.Cmd {
	cachePath := b.getCacheFilePath()
	file, err := os.Open(cachePath)
	if err == nil { // Si el archivo de caché existe
		defer file.Close()
		bytes, _ := io.ReadAll(file)
		var entry cacheEntry
		if json.Unmarshal(bytes, &entry) == nil {
			// Si el parseo JSON es exitoso y el timestamp es válido...
			if time.Since(entry.Timestamp) < b.cacheDuration {
				b.parsedData = entry.ParsedData // ¡Cargamos desde la caché!
				return nil                      // Y no hacemos nada más.
			}
		}
	}

    b.isLoading = true // Activar el spinner
	// Si la caché ha expirado, procede con la ejecución normal
	return func() tea.Msg {
		cmd := exec.Command(b.command, b.args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
            // CORRECCIÓN 4 (Parte A): Incluimos el ID en el mensaje de error.
			return dataMsg{blockID: b.id, err: fmt.Errorf("falló la ejecución del comando: %w", err)}
		}

		parsedData, err := b.parser.Parse(string(output))
		if err != nil {
            // CORRECCIÓN 4 (Parte B): Incluimos el ID en el mensaje de error.
			return dataMsg{blockID: b.id, err: fmt.Errorf("falló el parseo: %w", err)}
		}
		
        // CORRECCIÓN 4 (Parte C): Incluimos el ID en el mensaje de éxito.
		return dataMsg{blockID: b.id, data: parsedData}
	}
}

func (b *ShellCommandBlock) View() string {
	// Si está cargando, muestra el spinner alineado con el ID del bloque.
	if b.isLoading {
		// Usamos JoinHorizontal para alinear el spinner y el texto.
		spinnerView := b.spinner.View()
		idView := b.style.Copy().Faint(true).Render(b.id) // Atenuamos el ID
		return lipgloss.JoinHorizontal(lipgloss.Left, spinnerView, " ", idView)
	}

	// Si hay un error, lo mostramos usando el estilo base del bloque.
	if b.currentError != nil {
		// Aplicamos el estilo para asegurar consistencia de color y padding.
		errorMsg := fmt.Sprintf("Error en '%s': %v", b.id, b.currentError)
		return b.style.Copy().Foreground(lipgloss.Color("9")).Render(errorMsg) // Color rojo para errores
	}

	// Si no hay datos (y no está cargando), devuelve un string vacío.
	if b.parsedData == nil {
		return ""
	}

	// Si todo está bien, renderiza los datos.
	return b.renderer.Render(b.parsedData, 0, b.style)
}

// getCacheFilePath es una función helper para obtener la ruta del archivo de caché.
func (b *ShellCommandBlock) getCacheFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", b.id))
}

func (b *ShellCommandBlock) HandleMsg(msg tea.Msg) {
    // CORRECCIÓN 5: Comprobamos si el mensaje es un dataMsg Y si su ID coincide con el nuestro.
	if m, ok := msg.(dataMsg); ok && m.blockID == b.id {
        b.isLoading = false // Desactivar el spinner
		b.parsedData = m.data
		b.currentError = m.err

		// Si la actualización fue exitosa, escribimos en la caché.
		if m.err == nil {
			entry := cacheEntry{
				Timestamp:  time.Now(),
				ParsedData: m.data,
			}
			bytes, err := json.Marshal(entry)
			if err == nil {
				os.WriteFile(b.getCacheFilePath(), bytes, 0644)
			}
		}
	}
}
