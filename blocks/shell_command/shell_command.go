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

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gas/fancy-welcome/config"
	"github.com/gas/fancy-welcome/blocks/shell_command/parsers"
	"github.com/gas/fancy-welcome/blocks/shell_command/renderers"
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
// Mensaje para cuando los datos vienen de la caché
type cachedDataMsg struct {
	blockID string
	data    interface{}
	err     error
}

var registeredParsers = make(map[string]parsers.Parser)
var registeredRenderers = make(map[string]renderers.Renderer)

func init() {
	// Register Parsers
	registeredParsers["single_line"] = &parsers.SingleLineParser{}
	registeredParsers["multi_line"] = &parsers.MultiLineParser{}
	registeredParsers["app_count"] = &parsers.AppCountParser{}
	registeredParsers["dev_versions"] = &parsers.DevVersionsParser{}
	registeredParsers["journald_errors"] = &parsers.JournaldErrorsParser{}
	registeredParsers["key_value"] = &parsers.KeyValueParser{}
	
	// Register Renderers
	registeredRenderers["raw_text"] = &renderers.RawTextRenderer{}
	registeredRenderers["cowsay"] = &renderers.CowsayRenderer{}
	registeredRenderers["table"] = &renderers.TableRenderer{}
	registeredRenderers["gauge"] = &renderers.GaugeRenderer{}
	registeredRenderers["list"] = &renderers.ListRenderer{}
}

// Nuevo struct para guardar en el archivo de caché
type cacheEntry struct {
	Timestamp  time.Time   `json:"timestamp"`
	ParsedData interface{} `json:"parsed_data"`
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
    lastUpdateTime	time.Time
    isLoading 		bool
    spinner   		spinner.Model
    width 			int
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

func (b *ShellCommandBlock) SetWidth(width int) {
	b.width = width
}

func (b *ShellCommandBlock) Name() string {
    // Usamos el id como el nombre, ya que es único.
	return b.id
}

func (b *ShellCommandBlock) Spinner() *spinner.Model { return &b.spinner }

func (b *ShellCommandBlock) SpinnerCmd() tea.Cmd { return b.spinner.Tick }

func (b *ShellCommandBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, style lipgloss.Style) error {
	// 1 --- Inicialización básica.
	b.id = blockConfig["name"].(string)
	b.style = style
	logging.Log.Printf("[%s] Initializing block...", b.id)

	// 2. Lógica de comando menos enrevesada
	b.command, _ = blockConfig["command"].(string)
	logging.Log.Printf("[%s] Config loaded. Command: '%s'", b.id, b.command)	

	// 3. Lógica de Caché Simplificada
	// Se busca la clave "cache". Si no existe o es 0, la caché se deshabilita (duration = 0).
	if cacheSecs, ok := blockConfig["cache"].(float64); ok && cacheSecs > 0 {
		b.cacheDuration = time.Duration(cacheSecs) * time.Second
	} else {
		b.cacheDuration = 0 // Caché desactivada por defecto si no se especifica o es 0
	}
	logging.Log.Printf("[%s] Cache duration set to %v", b.id, b.cacheDuration)

	// --- 5. Inicialización de Parser y Renderer ---
	parserName, _ := blockConfig["parser"].(string)
	p, ok := registeredParsers[parserName]
	if !ok {
		return fmt.Errorf("parser '%s' no encontrado para el bloque '%s'", parserName, b.id)
	}
	b.parser = p

	rendererName, _ := blockConfig["renderer"].(string)
	r, ok := registeredRenderers[rendererName]
	if !ok {
		return fmt.Errorf("renderer '%s' no encontrado para el bloque '%s'", rendererName, b.id)
	}
	b.renderer = r

	// -- 6. spinner
    b.spinner = spinner.New()
    // Podemos estilizar el spinner usando los colores del tema
    b.spinner.Style = lipgloss.NewStyle().Foreground(style.GetForeground())

	return nil
}

func (b *ShellCommandBlock) Update() tea.Cmd {
	// La lógica de 'update_seconds' se ha movido al MainModel.

	if b.cacheDuration > 0 {
		cachePath := b.getCacheFilePath()
		file, err := os.Open(cachePath)
		if err == nil { // Si el archivo de caché existe
			defer file.Close()
			bytes, _ := io.ReadAll(file)
			var entry cacheEntry
			if json.Unmarshal(bytes, &entry) == nil {
				// Si el parseo JSON es exitoso y el timestamp es válido...
				if time.Since(entry.Timestamp) < b.cacheDuration {
					// LOGGING: Registra que se ha encontrado una caché válida
					logging.Log.Printf("[%s] CACHE HIT. Data is fresh.", b.id)
					// Se devuelve un mensaje para notificar que se usarán datos de la caché
					return func() tea.Msg {
						return cachedDataMsg{blockID: b.id, data: entry.ParsedData, err: nil}
					}
				}
			}
		}
	}

	// Si la caché está desactivada (cache=0 o no definido) O si está activada pero ha expirado,
	// se considera un CACHE MISS y se procede a ejecutar el comando.
	logging.Log.Printf("[%s] CACHE MISS or disabled. Preparing to execute command.", b.id)
   	b.isLoading = true // Activar el spinner

	// Si la caché ha expirado, procede con la ejecución normal
	return func() tea.Msg {
		var output []byte
		var err error

		// Ejecución de Comando Simplificada
		// Se usa 'sh -c' para permitir tuberías y otras operaciones de shell directamente.
		if b.command != "" {
			logging.Log.Printf("[%s] Executing command: sh -c \"%s\"", b.id, b.command)
			cmd := exec.Command("sh", "-c", b.command)
			output, err = cmd.CombinedOutput()
			if err != nil {
				logging.Log.Printf("[%s] EXECUTION ERROR: %v. Output: %s", b.id, err, string(output))
				return freshDataMsg{blockID: b.id, err: fmt.Errorf("falló la ejecución del comando: %w", err)}
			}
			// LOGGING: Registra la salida cruda del comando
			logging.Log.Printf("[%s] Raw command output: %s", b.id, strings.TrimSpace(string(output)))
		} else {
			// Esto es para parsers como app_count que no necesitan un comando externo
			logging.Log.Printf("[%s] No command to execute. Proceeding with parser.", b.id)
		}

		// Se parsea la salida del comando
		parsedData, err := b.parser.Parse(string(output))
		if err != nil {
			// LOGGING: Registra un error en el parseo
			logging.Log.Printf("[%s] PARSING ERROR: %v", b.id, err)			
            // Incluimos el ID en el mensaje de error.
			return freshDataMsg{blockID: b.id, err: fmt.Errorf("falló el parseo: %w", err)}
		}
		
		// LOGGING: Registra que el parseo fue exitoso
		logging.Log.Printf("[%s] Parsing successful.", b.id)
	    // Incluimos el ID en el mensaje de éxito.
		return freshDataMsg{blockID: b.id, data: parsedData}
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
		return b.style.Render("...") // Show ellipsis instead of nothing	
	}

	// Si todo está bien, renderiza los datos.
	return b.renderer.Render(b.parsedData, b.width, b.style)
}

// getCacheFilePath es una función helper para obtener la ruta del archivo de caché.
func (b *ShellCommandBlock) getCacheFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", b.id))
}

func (b *ShellCommandBlock) HandleMsg(msg tea.Msg) {

	handleCompletion := func() {
		b.isLoading = false
		b.lastUpdateTime = time.Now()
		logging.Log.Printf("[%s] Update cycle completed. New lastUpdateTime: %s", b.id, b.lastUpdateTime.Format(time.RFC3339))
	}

	// Usamos un switch para manejar los diferentes tipos de mensajes
	switch m := msg.(type) {
	// Caso para datos FRESCOS
	case freshDataMsg:
		if m.blockID == b.id {
			b.isLoading = false
			// llamaríamos a handleCompletion aquí, no está definida
			handleCompletion()

			b.currentError = m.err
			if m.err == nil {
				b.parsedData = m.data
				// Si la caché está activada, escribimos los nuevos datos.
				if b.cacheDuration > 0 {
					entry := cacheEntry{
						Timestamp:  time.Now(),
						ParsedData: m.data,
					}
					bytes, err := json.Marshal(entry)
					if err == nil {
						logging.Log.Printf("[%s] Writing new data to cache file: %s", b.id, b.getCacheFilePath())
						os.WriteFile(b.getCacheFilePath(), bytes, 0644)
					}
				}
			}
		}
	// Caso para datos de la CACHÉ
	case cachedDataMsg:
		if m.blockID == b.id {
			b.isLoading = false
			// llamaríamos a handleCompletion aquí?, no está definida
			handleCompletion()
			b.currentError = m.err
			if m.err == nil {
				b.parsedData = m.data
				// NO escribimos en la caché aquí para no resetear el timestamp
				logging.Log.Printf("[%s] Updated block with cached data.", b.id)
			}
		}
	}
}
