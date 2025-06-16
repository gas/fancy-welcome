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
	"github.com/gas/fancy-welcome/themes"
	"github.com/gas/fancy-welcome/logging" // paquete de logging
	"github.com/gas/fancy-welcome/shared/block"
    "github.com/gas/fancy-welcome/shared/messages" // Importa el nuevo paquete
)



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
   	updateInterval 	time.Duration
	nextRunTime 	time.Time
    isLoading 		bool
    spinner   		spinner.Model
    width 			int
   	position string // "left", "right", o vacío (full width)
    blockTheme *themes.Theme // Nuevo campo para almacenar el tema
	renderedHeight       int  // Altura del bloque en su última renderización
    contentChangedSinceLastView bool // Nuevo flag
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

func (b *ShellCommandBlock) Init(blockConfig map[string]interface{}, globalConfig config.GeneralConfig, theme *themes.Theme) error {
	// 1 --- Inicialización básica.
	b.id, _ = blockConfig["name"].(string)
	// Creamos un estilo base para el bloque usando los colores del tema.
	b.style = lipgloss.NewStyle().
		Background(lipgloss.Color(theme.Colors.Background)).
		Foreground(lipgloss.Color(theme.Colors.Text))

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

	// 4. Lógica de Tiempo de Actualización
	// Se busca "update_seconds" en el bloque. Si no se encuentra, se usa el valor global.
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

	// Se establece un valor mínimo de 1 segundo para evitar bucles infinitos.
	if updateSecs < 1 {
		updateSecs = 1
	}
	b.updateInterval = time.Duration(updateSecs) * time.Second
	logging.Log.Printf("[%s] Update interval set to %v", b.id, b.updateInterval)

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

	// --- Lógica del Spinner Corregida ---

	// 1. Leemos el estilo de indicador deseado desde la configuración del bloque
	indicatorStyle, _ := blockConfig["loading_indicator"].(string)
	if indicatorStyle == "" {
		indicatorStyle = "spinner" // Usamos 'spinner' como valor por defecto
	}
 
	// 2. Creamos el spinner con los fotogramas del tema o usamos uno por defecto
	spinnerOptions := []spinner.Option{
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color(theme.Colors.Primary))),
	}

	if style, ok := theme.Indicators[indicatorStyle]; ok && len(style.Frames) > 0 {
		spinnerAnimation := spinner.Spinner{Frames: style.Frames, FPS: time.Second / 10}
		spinnerOptions = append(spinnerOptions, spinner.WithSpinner(spinnerAnimation))
	}

	b.spinner = spinner.New(spinnerOptions...)

	// 3. posición left 50%, right 50% o full 100%
	if pos, ok := blockConfig["position"].(string); ok {
		b.position = pos
	} else {
		b.position = "full" // Valor por defecto
	}
    b.blockTheme = theme // Almacenar el tema

    b.nextRunTime = time.Now()
	return nil
}

// funciones de ancho y posición
func (b *ShellCommandBlock) GetPosition() string {
	return b.position
}

func (b *ShellCommandBlock) GetSetWidth() int { // Implementar el nuevo método
    return b.width
}

func (b *ShellCommandBlock) GetThemeColors() themes.ThemeColors {
    return b.blockTheme.Colors
}

func (b *ShellCommandBlock) Update() tea.Cmd {
	// Control de Frecuencia de Actualización (Limitador de frecuencia)
	// Si el bloque ya está cargando datos, no hacer nada.
	// Esto previene las ejecuciones solapadas (re-entrada).
	if b.isLoading {
		return nil
	}

	// Comprueba si ya es hora de ejecutar según el horario objetivo.
	//3 opciones, cada una tiene sus problemas, usamos la más robusta
	//if time.Since(b.lastUpdateTime) < b.updateInterval { // estricto, requiere un scheduler en main.go
	//if time.Since(b.lastUpdateTime).Round(time.Second) < b.updateInterval { // se acumula el desfase
	if time.Now().Before(b.nextRunTime) {
        if b.isLoading { // Si el comando ya está en progreso y no ha terminado
            return nil
        }
		return nil 
	}

    logging.Log.Printf("[%s] ACTIVANDO SPINNER: isLoading = true", b.id)
    b.isLoading = true // Activar el spinner

    
	// Comprobación de Caché Simplificada
	// Solo se intenta leer la caché si se ha definido una duración (cache > 0).
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
						return messages.CachedDataMsg{BlockID: b.id, Data: entry.ParsedData, Err: nil}
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
    // Batch el comando de ejecución y el comando para iniciar el tick del spinner
    //return tea.Batch(func() tea.Msg {
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
				return messages.FreshDataMsg{BlockID: b.id, Err: fmt.Errorf("falló la ejecución del comando: %w", err)}
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
			return messages.FreshDataMsg{BlockID: b.id, Err: fmt.Errorf("falló el parseo: %w", err)}
		}
		
		// LOGGING: Registra que el parseo fue exitoso
		logging.Log.Printf("[%s] Parsing successful.", b.id)
	    // Incluimos el ID en el mensaje de éxito.
		return messages.FreshDataMsg{BlockID: b.id, Data: parsedData}
	}//, b.spinner.Tick) // <-- Inicia el tick del spinner
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

    // Calcular la altura después de renderizar el contenido.
    // Asegúrate de que `renderedContent` es el string final incluyendo título y borde.
    // Para esto, necesitarás que `renderStyledBlock` (o similar) sea parte del View del bloque,
    // o que `View` devuelva el contenido *sin* el borde/título y la altura se calcule después.
    // Por simplicidad, aquí asumimos `blockView` es lo que genera `b.View()` antes del borde/título.
    //blockView := content // ... (el string que b.View() normalmente devuelve)
    //currentHeight := strings.Count(blockView, "\n") + 1 // Contar líneas, +1 para la última línea sin \n

	return content
}

func (b *ShellCommandBlock) RenderedHeight() int {
    if b.parsedData == nil {
        return 1 // O alguna altura por defecto para "Loading..."
    }
    // Renderizar el contenido para calcular su altura real con el ancho asignado, lipgloss no tiene estos métodos? o nuestra versión
    //renderedContent := b.renderer.Render(b.parsedData, b.width, b.style.Copy().UnsetMargins().UnsetPadding().UnsetBorder())
    renderedContent := b.renderer.Render(b.parsedData, b.width, b.style)
    return strings.Count(renderedContent, "\n") + 1
}

// Nuevo: Se activa cuando los datos REALMENTE cambian.
func (b *ShellCommandBlock) HasContentChanged() bool {
    return b.contentChangedSinceLastView
}

// Nuevo: Se resetea después de que el dashboard lo ha procesado.
func (b *ShellCommandBlock) ResetContentChangedFlag() {
    b.contentChangedSinceLastView = false
}

// getCacheFilePath es una función helper para obtener la ruta del archivo de caché.
func (b *ShellCommandBlock) getCacheFilePath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", b.id))
}

func (b *ShellCommandBlock) HandleMsg(msg tea.Msg) {
	// La lógica de HandleMsg se actualiza para guardar el tiempo de la última actualización
	handleCompletion := func() {
        if b.isLoading { // Solo log si estaba cargando y ahora se desactiva
            logging.Log.Printf("[%s] DESACTIVANDO SPINNER: isLoading = false", b.id)
            b.isLoading = false
        }
	}

	// Usamos un switch para manejar los diferentes tipos de mensajes
	switch m := msg.(type) {
	// Caso para datos FRESCOS
	case messages.FreshDataMsg:
		if m.BlockID == b.id {
			// Llamamos a handleCompletion para registrar la hora y detener la carga.
			//handleCompletion() 

			b.currentError = m.Err
			if m.Err == nil {
				b.parsedData = m.Data
				// Si la caché está activada, escribimos los nuevos datos.
				if b.cacheDuration > 0 {
					// Solo escribimos en la caché cuando los datos son frescos
					entry := cacheEntry{
						Timestamp:  time.Now(),
						ParsedData: m.Data,
					}
					bytes, err := json.Marshal(entry)
					if err == nil {
						logging.Log.Printf("[%s] Writing new data to cache file: %s", b.id, b.getCacheFilePath())
						os.WriteFile(b.getCacheFilePath(), bytes, 0644)
					}
				}
			}
			// Planifica la siguiente ejecución SOLO después de que ESTE bloque haya terminado.
			b.nextRunTime = b.nextRunTime.Add(b.updateInterval)

            if m.Err == nil { // Solo si los datos se actualizaron correctamente
                b.contentChangedSinceLastView = true // Marcar que el contenido ha cambiado
            }

			handleCompletion()
		}
	// Caso para datos de la CACHÉ
	case messages.CachedDataMsg:
		if m.BlockID == b.id {
			// También llamamos a handleCompletion para registrar la hora de la actualización.
			//handleCompletion()

			b.currentError = m.Err
			if m.Err == nil {
				b.parsedData = m.Data
				logging.Log.Printf("[%s] Updated block with cached data.", b.id)
			}

			// Planifica la siguiente ejecución SOLO después de que ESTE bloque haya terminado.
			b.nextRunTime = b.nextRunTime.Add(b.updateInterval)

            // Opcional: Marcar como cambiado si la caché cambia, aunque es menos común.
            // b.contentChangedSinceLastView = true

			handleCompletion()
		}
	}
}
