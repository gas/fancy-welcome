// shared/messages/messages.go
package messages

import "time" // Necesario si usas time.Time en los mensajes

// FreshDataMsg es un mensaje enviado cuando los datos de un bloque son nuevos (de un comando o cálculo).
type FreshDataMsg struct {
	BlockID string
	Data    interface{}
	Err     error
}

// CachedDataMsg es un mensaje enviado cuando los datos de un bloque vienen de la caché.
type CachedDataMsg struct {
	BlockID string
	Data    interface{}
	Err     error
}

// InfoMsg es un mensaje enviado cuando la información de SystemInfoBlock es obtenida.
type InfoMsg struct {
	BlockID string
	Info    string
}

// TickMsg es el mensaje genérico de un "tick" global, si lo necesitas.
// Puedes usar el spinner.TickMsg de Bubble Tea si eso es suficiente.
// Si tuvieras un Tick global personalizado, iría aquí.
type PeriodicUpdateMsg time.Time // Renombrado a lo que ya usabas para consistencia