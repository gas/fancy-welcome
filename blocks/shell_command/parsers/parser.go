// blocks/shell_command/parsers/parser.go
package parsers

// Parser es la interfaz que cada módulo de parseo debe implementar.
// Toma un string de entrada y lo transforma en datos estructurados.
type Parser interface {
    // Parse toma el texto crudo y devuelve los datos parseados o un error.
    // Usamos interface{} para que cada parser pueda devolver el tipo de dato
    // que más le convenga (string, []string, map[string]string, etc.).
    Parse(input string) (interface{}, error)
}
