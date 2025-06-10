// blocks/shell_command/parsers/app_count.go
package parsers

import (
	"bytes"
	"os/exec"
	"strings"
)

type AppCountParser struct{}

func (p *AppCountParser) Parse(input string) (interface{}, error) {
	// This parser ignores the input and runs its own commands.
	
	// Helper function to run a command and count lines
	countLines := func(command string) string {
		cmd := exec.Command("sh", "-c", command)
		var out bytes.Buffer
		cmd.Stdout = &out
		err := cmd.Run()
		if err != nil {
			// If a package manager isn't installed, it's not an error, just means 0 packages.
			return "0"
		}
		// Subtract 1 to account for the header line in many list commands
		count := len(strings.Split(strings.Trim(out.String(), "\n"), "\n"))
		if count > 0 {
			return string(rune(count-1))
		}
		return "0"
	}

	snapCount := countLines("snap list")
	aptCount := countLines("apt list --installed 2>/dev/null")
	// You could add flatpak, etc. here
	// flatpakCount := countLines("flatpak list")

	data := [][]string{
		{"Package Manager", "Count"},
		{"snap", snapCount},
		{"apt", aptCount},
		// {"flatpak", flatpakCount},
	}

	return data, nil
}