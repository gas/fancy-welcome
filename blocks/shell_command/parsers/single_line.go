// blocks/shell_command/parsers/single_line.go
package parsers

import "strings"

type SingleLineParser struct{}

func (p *SingleLineParser) Parse(input string) (interface{}, error) {
    return strings.TrimSpace(input), nil
}
