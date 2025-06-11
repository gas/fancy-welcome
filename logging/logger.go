// logging/logger.go
package logging

import (
	"log"
	"os"
)

var (
	// Log es nuestro logger global que usaremos en toda la aplicación.
	Log *log.Logger
)

func Init() (*os.File, error) {
	// Abrimos (o creamos) un archivo de log.
	// Usar /tmp/ es una forma sencilla de asegurarse de que se puede escribir.
	logFile, err := os.OpenFile("/tmp/fancy-welcome.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0664)
	if err != nil {
		return nil, err
	}

	// Creamos una nueva instancia de logger que escribirá en nuestro archivo.
	Log = log.New(logFile, "fancy-welcome: ", log.LstdFlags|log.Lshortfile)
	Log.Println("Logging system initialized.")

	return logFile, nil
}