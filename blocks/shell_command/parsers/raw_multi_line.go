// blocks/shell_command/parsers/raw_multi_line.go
package parsers

import "strings"

type RawMultiLineParser struct{}

func (p *RawMultiLineParser) Parse(input string) (interface{}, error) {
    // Divide por nueva línea y no elimina espacios en blanco de cada una.
    lines := strings.Split(input, "\n")
    var cleanedLines []string
    for _, line := range lines {
        if strings.TrimSpace(line) != "" {
            cleanedLines = append(cleanedLines, line)
        }
    }
    return cleanedLines, nil
}
