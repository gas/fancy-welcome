// blocks/shell_command/shell_command.go
package shell_command

import (
	//"encoding/json"	
	"fmt"
	//"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/blocks/shell_command/parsers"
	"github.com/gas/fancy-welcome/blocks/shell_command/renderers"
	"github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/logging" // paquete de logging
	"github.com/gas/fancy-welcome/shared/block"
)


// --- NUEVOS TIPOS DE MENSAJE ---
// Mensaje para cuando los datos son nuevos (de un comando)
type freshDataMsg struct {
	blockID string
	data    interface{}
	err     error
}

func (m freshDataMsg) BlockID() string { return m.blockID } // <-- AÑADE ESTE MÉTODO

// Mensaje para cuando los datos vienen de la caché
type cachedDataMsg struct {
	blockID string
	data    interface{}
	err     error
}

func (m cachedDataMsg) BlockID() string { return m.blockID } // <-- AÑADE ESTE MÉTODO

// Nuevo struct para guardar en el archivo de caché
type cacheEntry struct {
	Timestamp  time.Time   `json:"timestamp"`
	ParsedData interface{} `json:"parsed_data"`
}


var registeredParsers = make(map[string]parsers.Parser)
var registeredRenderers = make(map[string]renderers.Renderer)

func init() {
	// Register Parsers
	registeredParsers["single_line"] = &parsers.SingleLineParser{}
	registeredParsers["multi_line"] = &parsers.MultiLineParser{}
	registeredParsers["raw_multi_line"] = &parsers.RawMultiLineParser{}
	registeredParsers["app_count"] = &parsers.AppCountParser{}
	registeredParsers["dev_versions"] = &parsers.DevVersionsParser{}
	registeredParsers["journald_errors"] = &parsers.JournaldErrorsParser{}
	registeredParsers["key_value"] = &parsers.KeyValueParser{}
	registeredParsers["raw_text"] = &parsers.RawTextParser{} 

	// Register Renderers
	registeredRenderers["raw_text"] = &renderers.RawTextRenderer{}
	registeredRenderers["cowsay"] = &renderers.CowsayRenderer{}
	registeredRenderers["table"] = &renderers.TableRenderer{}
	registeredRenderers["gauge"] = &renderers.GaugeRenderer{}
	registeredRenderers["list"] = &renderers.ListRenderer{}
	registeredRenderers["raw_list"] = &renderers.RawListRenderer{}
	registeredRenderers["preformatted_text"] = &renderers.PreformattedTextRenderer{}

}

// 1: Añadido el campo 'id' al struct del bloque.
type ShellCommandBlock struct {
	id           	string // ID único del bloque
	style        	lipgloss.Style
	command      	string
	parser       	parsers.Parser
	renderer     	renderers.Renderer
	parsedData   	interface{}
	currentError 	error
    cacheDuration 	time.Duration // 0 significa que la caché está desactivada
   	updateInterval 	time.Duration
	//nextRunTime 	time.Time
    isLoading 		bool
    spinner   		spinner.Model
    position     string
    width 			int
    blockConfig    map[string]interface{}
}

// CORRECCIÓN 2: Añadir 'blockID' al mensaje para saber a quién pertenece.
type dataMsg struct {
	blockID string
	data    interface{}
	err     error
}

func (b *ShellCommandBlock) Position() string {
    return b.position
}

func New() block.Block {
	return &ShellCommandBlock{}
}

func (b *ShellCommandBlock) SetWidth(width int) {
	b.width = width
}

func (b *ShellCommandBlock) Name() string {
    // Usamos el id como el nombre, ya que es único.
	return b.id
}

// Helper para obtener la ruta del archivo de caché (duplicado de shell_command para uso en main)
func (b *ShellCommandBlock) getCacheFilePath() string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", b.id))
}

func (b *ShellCommandBlock) Spinner() *spinner.Model { return &b.spinner }

func (b *ShellCommandBlock) SpinnerCmd() tea.Cmd { return b.spinner.Tick }

// Reemplaza tu función Init en shell_command.go con esta:
func (b *ShellCommandBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error {
	b.blockConfig = blockConfig
	b.id, _ = blockConfig["name"].(string)
	b.position, _ = blockConfig["position"].(string)
	b.style = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Text))
	logging.Log.Printf("[%s] Initializing block...", b.id)

	// --- LÓGICA DE DEPURACIÓN DEL INTERVALO ---
	//logging.Log.Printf("[%s] --- Start Interval Calculation ---", b.id)
	
	var updateSecs float64 = 0
	val, ok := blockConfig["update_seconds"]
	//logging.Log.Printf("[%s] 1. Reading 'update_seconds' -> Found: %v, Ok: %v, Type: %T", b.id, val, ok, val)

	if ok {
		switch v := val.(type) {
		case float64:
			updateSecs = v
		case int:
			updateSecs = float64(v)
		case int64:
			updateSecs = float64(v)
		}
	}
	//logging.Log.Printf("[%s] 2. 'updateSecs' after parsing from config: %.2f", b.id, updateSecs)

	if updateSecs <= 0 {
		updateSecs = globalConfig.GlobalUpdateSeconds
		//logging.Log.Printf("[%s] 3. 'updateSecs' is <= 0, falling back to global: %.2f", b.id, updateSecs)
	}

	if updateSecs < 1 {
		updateSecs = 1
		//logging.Log.Printf("[%s] 4. 'updateSecs' is < 1, forcing to minimum: %.2f", b.id, updateSecs)
	}
	
	b.updateInterval = time.Duration(updateSecs) * time.Second
	//logging.Log.Printf("[%s] 5. Final 'updateInterval' set to: %v", b.id, b.updateInterval)
	//logging.Log.Printf("[%s] --- End Interval Calculation ---", b.id)
	// --- FIN DE LA LÓGICA DE DEPURACIÓN ---

	b.command, _ = blockConfig["command"].(string)
	if cacheSecs, ok := blockConfig["cache"].(float64); ok && cacheSecs > 0 {
		b.cacheDuration = time.Duration(cacheSecs) * time.Second
	} else {
		b.cacheDuration = 0
	}

	parserName, _ := blockConfig["parser"].(string)
	b.parser = registeredParsers[parserName]
	rendererName, _ := blockConfig["renderer"].(string)
	b.renderer = registeredRenderers[rendererName]
	indicatorStyle, _ := blockConfig["loading_indicator"].(string)
	if indicatorStyle == "" {
		indicatorStyle = "spinner"
	}
	spinnerOptions := []spinner.Option{
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Primary))),
	}
	if style, ok := theme.Indicators[indicatorStyle]; ok && len(style.Frames) > 0 {
		spinnerAnimation := spinner.Spinner{Frames: style.Frames, FPS: time.Second / 10}
		spinnerOptions = append(spinnerOptions, spinner.WithSpinner(spinnerAnimation))
	}
	b.spinner = spinner.New(spinnerOptions...)
	return nil
}


// En shell_command.go Y en system_info.go
func (b *ShellCommandBlock) Update(msg tea.Msg) (block.Block, tea.Cmd) {
    //logging.Log.Printf("SC Update: [%s] received msg: %T", b.id, msg)

    var cmd tea.Cmd

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

		// (El resto de la lógica de caché y fetchDataCmd se mantiene igual)
		// ...
		b.isLoading = true
		//return b, b.fetchDataCmd() // O el comando que sea para system_info
		return b, tea.Batch(
			b.fetchDataCmd(), // El comando para cargar los datos
			b.spinner.Tick,   // El comando para INICIAR la animación del spinner
		)

	// Cuando los datos llegan, programamos el SIGUIENTE TICK DIRIGIDO.
	case freshDataMsg: // O infoMsg para system_info
		if m.BlockID() != b.id { return b, nil }
		b.isLoading = false
		b.parsedData = m.data // o b.info = m.info
		
		// Llamamos al nuevo planificador dirigido.
		return b, block.ScheduleNextTick(b.id, b.updateInterval)

	case cachedDataMsg:
		if m.BlockID() != b.id { return b, nil }
		b.isLoading = false
		b.parsedData = m.data
		
		// También usamos el nuevo planificador.
		return b, block.ScheduleNextTick(b.id, b.updateInterval)

	case spinner.TickMsg:
        if b.isLoading {
            b.spinner, cmd = b.spinner.Update(m)
	        return b, cmd
        }
	}
	return b, nil
}

// fetchDataCmd es un método helper que devuelve el comando para la carga de datos.
func (b *ShellCommandBlock) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		var output []byte
		var err error

		// Se usa 'sh -c' para permitir tuberías y otras operaciones de shell. 
		if b.command != "" {
			//logging.Log.Printf("[%s] Executing command: sh -c \"%s\"", b.id, b.command)
			cmd := exec.Command("sh", "-c", b.command)
			output, err = cmd.CombinedOutput()
			if err != nil {
				logging.Log.Printf("[%s] EXECUTION ERROR: %v. Output: %s", b.id, err, string(output))
				// Devolvemos el error en el mensaje para que el bloque lo gestione.
				return freshDataMsg{blockID: b.id, err: fmt.Errorf("falló la ejecución: %w", err)}
			}
			logging.Log.Printf("[%s] Raw command output: %s", b.id, strings.TrimSpace(string(output)))
		} else {
			logging.Log.Printf("[%s] No command to execute. Proceeding with parser.", b.id)
		}

		// Se parsea la salida del comando. 
		parsedData, err := b.parser.Parse(string(output))
		if err != nil {
			logging.Log.Printf("[%s] PARSING ERROR: %v", b.id, err)
			// Devolvemos el error de parseo. 
			return freshDataMsg{blockID: b.id, err: fmt.Errorf("falló el parseo: %w", err)}
		}

		logging.Log.Printf("[%s] Parsing successful.", b.id)
		// Devolvemos los datos parseados con éxito. 
		return freshDataMsg{blockID: b.id, data: parsedData}
	}
}



func (b *ShellCommandBlock) View() string {
	var content string

	if b.currentError != nil {
		errorMsg := fmt.Sprintf("Error en '%s': %v", b.id, b.currentError)
		content = b.style.Copy().Foreground(lipgloss.Color("9")).Render(errorMsg)
	} else if b.parsedData != nil {
		// Si tenemos datos (antiguos o nuevos), los renderizamos.
		content = b.renderer.Render(b.parsedData, b.width, b.style)
	} else {
		// No hay datos ni error, probablemente la carga inicial.
		content = "..."
	}

	// Si está cargando, añadimos el spinner al final del contenido.
	if b.isLoading {
		return lipgloss.JoinHorizontal(lipgloss.Top, content, " "+b.spinner.View())
	}
	return content
}