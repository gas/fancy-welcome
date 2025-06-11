// config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type GeneralConfig struct {
	EnabledBlocksOrder []string `toml:"enabled_blocks_order"`
	GlobalUpdateSeconds float64  `toml:"global_update_seconds"` // Update time de la app
}

type ThemeConfig struct {
	SelectedTheme string `toml:"selected_theme"`
}

type Config struct {
	General GeneralConfig            `toml:"general"`
	Theme   ThemeConfig              `toml:"theme"`
	Blocks  map[string]interface{}   `toml:"blocks"`
}

func LoadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener el directorio home: %w", err)
	}

	configPath := filepath.Join(homeDir, ".config", "fancy-welcome", "fancy_welcome.toml")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el archivo de configuración %s: %w", configPath, err)
	}

	var cfg Config
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("no se pudo parsear el TOML de configuración: %w", err)
	}

	return &cfg, nil
}
