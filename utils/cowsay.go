// utils/cowsay.go
package utils

import (
    "fmt"
    "strings"
)

// Generate crea un mensaje de cowsay simple.
func Generate(message string, width int) string {
    border := strings.Repeat("-", len(message)+2)
    return fmt.Sprintf(" <%s>\n  %s\n   \\\n    \\\n        .--.\n       |o_o |\n       |:_/ |\n      //   \\ \\\n     (|     | )\n    /'\\_   _/`\\\n    \\___)=(___/", message, border)
}
