// main.go
package main

import (
	//"flag"
	//"fmt"
	"log"
	"os"
	//"path/filepath"

	//"github.com/charmbracelet/bubbles/viewport"
    //"github.com/charmbracelet/bubbles/textinput"	
	//"github.com/charmbracelet/bubbletea"
	//"github.com/charmbracelet/lipgloss"
	//"github.com/mattn/go-isatty"
	"github.com/urfave/cli/v2"

	//"github.com/gas/fancy-welcome/blocks/shell_command"
	//"github.com/gas/fancy-welcome/blocks/system_info"
	//"github.com/gas/fancy-welcome/blocks/word_counter"
    //"github.com/gas/fancy-welcome/blocks/filter"
	//"github.com/gas/fancy-welcome/config"
	//"github.com/gas/fancy-welcome/shared/block"
	//"github.com/gas/fancy-welcome/themes"

	"github.com/gas/fancy-welcome/logging"
	"github.com/gas/fancy-welcome/modes" 
)

func main() {
	// Inicializamos el logger globalmente 
	logFile, err := logging.Init()
	if err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	defer logFile.Close() // Nos aseguramos de cerrar el archivo al salir 

	// Definimos el flag para la salida simple/TTY, para poder reutilizarlo
	simpleOutputFlag := &cli.BoolFlag{
		Name:    "simple",
		Aliases: []string{"s"},
		Usage:   "Muestra la salida como texto plano sin TUI (modo TTY/volcado).",
	}

	app := &cli.App{
		Name:  "fancy-cli",
		Usage: "Una herramienta TUI modular y extensible",
		Commands: []*cli.Command{
			{
				Name:    "welcome",
				Aliases: []string{"w"},
				Usage:   "Muestra un saludo visual secuencial con bloques de información",
				Flags: []cli.Flag{
					simpleOutputFlag, // Flag para elegir salida TUI o TTY
					&cli.StringFlag{
						Name:  "refresh",
						Value: "",
						Usage: "Forzar refresco de un bloque específico o 'all' para todos.",
					},
				},
				Action: func(c *cli.Context) error {
					// La acción de 'welcome' decide qué renderizador usar
					if c.Bool("simple") {
						return modes.RunWelcomeTTY(c.String("refresh"))
					}
					return modes.RunWelcomeTUI(c.String("refresh"))
				},
			},
			{
				Name:    "filter",
				Aliases: []string{"f"},
				Usage:   "Visualiza datos y añade filtros dinámicamente",
				Flags: []cli.Flag{
					simpleOutputFlag, // El modo 'filter' también tiene la opción de salida simple
				},
				Action: func(c *cli.Context) error {
					// La acción de 'filter' hace lo mismo
					if c.Bool("simple") {
						return modes.RunFilterTTY()
					}
					return modes.RunFilterTUI()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logging.Log.Fatal(err)
	}
}