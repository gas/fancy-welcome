// blocks/shell_command/parsers/dev_versions.go
package parsers

import (
	"os/exec"
	"strings"
)

type DevVersionsParser struct{}

func (p *DevVersionsParser) Parse(input string) (interface{}, error) {
	// This parser also ignores input and runs its own set of commands.

	// Helper to run a command and get its version output
	getVersion := func(command ...string) string {
		cmd := exec.Command(command[0], command[1:]...)
		output, err := cmd.Output()
		if err != nil {
			return "Not found"
		}
		// Clean up the output
		version := strings.TrimSpace(string(output))
		version = strings.Replace(version, "go version go", "", 1)
		version = strings.Replace(version, "Python ", "", 1)
		return version

	}

	nodeVersion := getVersion("node", "--version")
	pythonVersion := getVersion("python3", "--version")
	goVersion := getVersion("go", "version")

	data := [][]string{
		{"Tool", "Version"},
		{"Node.js", nodeVersion},
		{"Python", pythonVersion},
		{"Go", goVersion},
	}

	return data, nil
}