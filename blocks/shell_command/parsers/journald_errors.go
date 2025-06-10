// blocks/shell_command/parsers/journald_errors.go
package parsers

import (
	"fmt"
	"regexp"
	"strings"
)

type JournaldErrorsParser struct{}

func (p *JournaldErrorsParser) Parse(input string) (interface{}, error) {
	lines := strings.Split(input, "\n")
	var cleanedLines []string
	// Regex to capture the most common log format, might need adjustments
	logPattern := regexp.MustCompile(`^\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2}\s+[\w.-]+\s+([^:]+):\s+(.*)`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		matches := logPattern.FindStringSubmatch(trimmed)
		if len(matches) > 2 {
			// Format as "Process: Message"
			formattedLine := fmt.Sprintf("%s: %s", matches[1], matches[2])
			cleanedLines = append(cleanedLines, formattedLine)
		} else {
			cleanedLines = append(cleanedLines, trimmed) // Fallback to raw line
		}
	}
	return cleanedLines, nil
}