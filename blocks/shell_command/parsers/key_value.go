// blocks/shell_command/parsers/key_value.go
package parsers

import (
	"strings"
)

type KeyValueParser struct{}

// Parse expects input in the format "key1=value1\nkey2=value2"
func (p *KeyValueParser) Parse(input string) (interface{}, error) {
	data := make(map[string]string)
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) == 2 {
			data[parts[0]] = parts[1]
		}
	}
	return data, nil
}