// blocks/shell_command/parsers/raw_text.go
package parsers


type RawTextParser struct{}

func (p *RawTextParser) Parse(input string) (interface{}, error) {
    // Simplemente devuelve la entrada tal cual.
    return input, nil
}