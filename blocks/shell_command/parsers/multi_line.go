// blocks/shell_command/parsers/multi_line.go
package parsers

import "strings"

type MultiLineParser struct{}

func (p *MultiLineParser) Parse(input string) (interface{}, error) {
    // Divide por nueva línea y elimina espacios en blanco de cada una.
    lines := strings.Split(input, "\n")
    var cleanedLines []string
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if trimmed != "" {
            cleanedLines = append(cleanedLines, trimmed)
        }
    }
    return cleanedLines, nil
}
