// themes/themes.go
package themes

import (
	"fmt"
	"os"

	//"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
)

type ThemeColors struct {
	Primary    string `toml:"primary"`
	Secondary  string `toml:"secondary"`
	Background string `toml:"background"`
	Text       string `toml:"text"`
	Error      string `toml:"error"`
}

type Theme struct {
	Name   string      `toml:"name"`
	Colors ThemeColors `toml:"colors"`
}

func LoadTheme(themeName string) (*Theme, error) {
    // Por ahora, asumimos que los temas están en el mismo directorio que el ejecutable.
    // En el futuro, se podría buscar en ~/.config/fancy-welcome/themes/ también.
	filePath := fmt.Sprintf("themes/%s.toml", themeName)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el archivo de tema %s: %w", filePath, err)
	}

	var theme Theme
	err = toml.Unmarshal(data, &theme)
	if err != nil {
		return nil, fmt.Errorf("no se pudo parsear el TOML del tema: %w", err)
	}

	return &theme, nil
}
