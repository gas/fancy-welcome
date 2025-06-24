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
	"bufio"

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
    position     	string
    width 			int
	rendererName   	string 
    isStreaming    	bool // <-- STREAM 
    program      	*tea.Program // <-- ¡NUEVO CAMPO! Guardará el puntero.
   	blockConfig    	map[string]interface{}
}

// Añadido 'blockID' al mensaje para saber a quién pertenece.
type dataMsg struct {
	blockID string
	data    interface{}
	err     error
}

// STREAM
// streamLineMsg transporta una sola línea de un comando en modo streaming.
type streamLineMsg struct {
	blockID string
	line    string
}
// Hacemos que sea un mensaje dirigido
func (m streamLineMsg) BlockID() string { return m.blockID }


// streamClosedMsg notifica que un comando en modo streaming ha terminado o ha fallado.
type streamClosedMsg struct {
	blockID string
	err     error
}
func (m streamClosedMsg) BlockID() string { return m.blockID }

// SetProgram guarda la referencia al programa para uso en el streaming.
func (b *ShellCommandBlock) SetProgram(p *tea.Program) {
    b.program = p
}

// OTROS


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
    b.rendererName = rendererName // para pasarselo a main

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

    b.isStreaming, _ = blockConfig["streaming"].(bool) // <-- STREAM
	return nil
}

func (b *ShellCommandBlock) RendererName() string {
    return b.rendererName 
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
			if tick.BlockID() != b.id {
				return b, nil // No es para mí, lo ignoro.
			}
		}

		// Si llegamos aquí, es nuestro turno de actualizar.
		if b.isLoading { return b, nil }

        // --- LÓGICA DE DECISIÓN: ¿STREAMING O COMANDO NORMAL? ---
        if b.isStreaming {
            logging.Log.Printf("[%s] Starting stream...", b.id)
            b.isLoading = true // Mostramos el spinner mientras se conecta
            cmd := exec.Command("sh", "-c", b.command)
            // Devolvemos un nuevo tipo de comando que escucha el stream
			return b, listenToStream(b.program, cmd, b.id)
        } else {
			// COMANDO NORMAL
			b.isLoading = true
			// el Batch que muestra vivos los spinners
			return b, tea.Batch(
				b.fetchDataCmd(), // El comando para cargar los datos
				b.spinner.Tick,   // El comando para INICIAR la animación del spinner
			)
		}

    // --- GESTIÓN DE MENSAJES DE STREAM ---

    case streamClosedMsg:
        if m.blockID != b.id { return b, nil }
        b.isLoading = false
        b.currentError = m.err
        logging.Log.Printf("[%s] Closed stream...", b.id)

        // El stream ha muerto, programamos un reintento.
        return b, block.ScheduleNextTick(b.id, time.Second*5) // Reintentar en 5s

    // --- MENSAJES DE COMANDOS NORMALES ---

	// Cuando los datos llegan, programamos el SIGUIENTE TICK DIRIGIDO.
	case freshDataMsg: // O infoMsg para system_info
		if m.BlockID() != b.id { return b, nil }
		b.isLoading = false
		b.parsedData = m.data // o b.info = m.info
		b.currentError = m.err

		// Creamos el comando para emitir los datos
		teeCmd := func() tea.Msg {
			return block.TeeOutputMsg{
				SourceBlockID: b.id,
				Output:        b.parsedData,
			}
		}
		
		// Devolvemos AMBOS comandos: el del siguiente tick y el de la emisión "tee"
		return b, tea.Batch(
			block.ScheduleNextTick(b.id, b.updateInterval),
			teeCmd,
		)

	//Es necesario hacer broadcast de la caché? En principio no.
	case cachedDataMsg:
		if m.BlockID() != b.id { return b, nil }
		b.isLoading = false
		b.parsedData = m.data
		
		// También usamos el nuevo planificador.
		return b, block.ScheduleNextTick(b.id, b.updateInterval)

	case block.StreamLineBatchMsg:
			if m.BlockID() != b.id {
				return b, nil
			}
			
			// Una vez que recibimos el primer lote, consideramos que ya no está "cargando".
			b.isLoading = false

			// Para la propia vista del bloque tail_log, podemos mostrar la última línea recibida.
			if len(m.Lines) > 0 {
				b.parsedData = m.Lines[len(m.Lines)-1]
			}
			
			// Creamos UN SOLO TeeOutputMsg que contiene TODAS las líneas.
			teeCmd := func() tea.Msg {
				return block.TeeOutputMsg{
					SourceBlockID: b.id,
					Output:        m.Lines, // <-- El Output ahora es un []string
				}
			}
			
			// Devolvemos un único comando "tee".
			return b, teeCmd

	case spinner.TickMsg:
        if b.isLoading {
            b.spinner, cmd = b.spinner.Update(m)
	        return b, cmd
        }
	}
	return b, nil
}

// fetchDataCmd es un método helper que devuelve el comando para la carga de datos.
func (b *ShellCommandBlock) OLD_fetchDataCmd() tea.Cmd {
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

// fetchDataCmd si no necesita p*program
func (b *ShellCommandBlock) fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		// Lanzamos el trabajo pesado en una goroutine.

		var output []byte
		var err error

		if b.command != "" {
			cmd := exec.Command("sh", "-c", b.command)
			// CombinedOutput sigue siendo bloqueante, pero ahora dentro de la goroutine.
			output, err = cmd.CombinedOutput()
			if err != nil {
				// Cuando termina (con error), enviamos el resultado al programa.
				return freshDataMsg{blockID: b.id, err: fmt.Errorf("falló la ejecución: %w", err)}
			}
		}
		
		// Parseamos la salida.
		parsedData, err := b.parser.Parse(string(output))
		if err != nil {
			return freshDataMsg{blockID: b.id, err: fmt.Errorf("falló el parseo: %w", err)}
		}
		
		// Cuando termina (con éxito), enviamos el resultado al programa.
		return freshDataMsg{blockID: b.id, data: parsedData}	
	}
}

// fetchDataCmd antes era asíncrono para que no bloqueara.
func (b *ShellCommandBlock) FORMER_fetchDataCmd(p *tea.Program) tea.Cmd {
	return func() tea.Msg {
		// Lanzamos el trabajo pesado en una goroutine.
		go func() {
			var output []byte
			var err error

			if b.command != "" {
				cmd := exec.Command("sh", "-c", b.command)
				// CombinedOutput sigue siendo bloqueante, pero ahora dentro de la goroutine.
				output, err = cmd.CombinedOutput()
				if err != nil {
					// Cuando termina (con error), enviamos el resultado al programa.
					p.Send(freshDataMsg{blockID: b.id, err: err})
					return
				}
			}
			
			// Parseamos la salida.
			parsedData, err := b.parser.Parse(string(output))
			if err != nil {
				p.Send(freshDataMsg{blockID: b.id, err: err})
				return
			}
			
			// Cuando termina (con éxito), enviamos el resultado al programa.
			p.Send(freshDataMsg{blockID: b.id, data: parsedData})
		}()

		// La función principal devuelve 'nil' inmediatamente,
		// sin bloquear el bucle de Bubble Tea. La goroutine queda trabajando.
		return nil
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

// listenToStream crea un comando que inicia un proceso y escucha su salida
// línea por línea en una goroutine.
// blocks/shell_command/shell_command.go

func listenToStream(p *tea.Program, cmd *exec.Cmd, blockID string) tea.Cmd {
	return func() tea.Msg {
		stdout, err := cmd.StdoutPipe()
		if err != nil { return streamClosedMsg{blockID: blockID, err: err} }

		if err := cmd.Start(); err != nil {
			return streamClosedMsg{blockID: blockID, err: err}
		}

		scanner := bufio.NewScanner(stdout)
		
		go func() {
			// Buffer para agrupar líneas
			var lines []string
			// Temporizador para enviar lotes cada 100ms
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				// Caso 1: Ha pasado el tiempo del ticker
				case <-ticker.C:
					if len(lines) > 0 {
						// Si tenemos líneas acumuladas, las enviamos
						// p.Send(block.NewStreamLineBatchMsg{blockID: blockID, lines: lines})
						p.Send(block.NewStreamLineBatchMsg(blockID, lines))
						// Y vaciamos el buffer
						lines = nil
					}
				// Caso 2: Leemos una nueva línea del scanner
				default:
					if !scanner.Scan() {
						// Si el scanner falla o termina, el proceso ha muerto.
						// Enviamos las últimas líneas que quedaran y cerramos.
						if len(lines) > 0 {
							// p.Send(block.StreamLineBatchMsg{blockID: blockID, lines: lines})
							p.Send(block.NewStreamLineBatchMsg(blockID, lines))
						}
						p.Send(block.NewStreamClosedMsg(blockID, cmd.Wait()))
						return
					}
					// Acumulamos la línea leída en el buffer
					lines = append(lines, scanner.Text())
				}
			}
		}()

		return nil
	}
}
