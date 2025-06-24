// shared/setup.go
package shared

import (
    "fmt" // Necesitamos fmt para getCacheFilePath
    "os"
    "path/filepath"
    "log" // Necesitamos log para el Printf de error

    // Todos los paquetes necesarios para la inicialización
    "github.com/gas/fancy-welcome/config"
    //"github.com/gas/fancy-welcome/logging"
    "github.com/gas/fancy-welcome/shared/block"
    "github.com/gas/fancy-welcome/themes"
    "github.com/gas/fancy-welcome/blocks/shell_command"
    "github.com/gas/fancy-welcome/blocks/system_info"
    "github.com/gas/fancy-welcome/blocks/word_counter"
    "github.com/gas/fancy-welcome/blocks/filter"
)

// SetupResult agrupa todo lo que la inicialización produce.
type SetupResult struct {
    Config       *config.Config
    Theme        *themes.Theme
    ActiveBlocks []block.Block
    BlockFactory map[string]func() block.Block
}

// getCacheFilePath es una función helper que ahora vive aquí.
// La hacemos privada (minúscula) porque solo se usa dentro de este paquete.
func getCacheFilePath(blockName string) string {
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".cache", "fancy-welcome", fmt.Sprintf("%s.json", blockName))
}

// Setup realiza toda la carga de configuración e inicialización de bloques.
func Setup(refreshTarget string) (*SetupResult, error) {
    cfg, err := config.LoadConfig()
    if err != nil { return nil, err }

    theme, err := themes.LoadTheme(cfg.Theme.SelectedTheme)
    if err != nil { return nil, err }

    // Lógica de caché (tomada de tu main.go original)
    homeDir, _ := os.UserHomeDir()
    cacheDir := filepath.Join(homeDir, ".cache", "fancy-welcome")
    if err := os.MkdirAll(cacheDir, 0755); err != nil {
        return nil, err
    }

    // Fábrica de bloques (tomada de tu main.go original)
    blockFactory := map[string]func() block.Block{
        "ShellCommand": shell_command.New,
        "SystemInfo":   system_info.New,
        "WordCounter":  word_counter.New,
        "Filter":       filter.New,
    }

    var activeBlocks []block.Block
    for _, blockName := range cfg.General.EnabledBlocksOrder {
        // ... (Aquí va toda la lógica del bucle de tu runTuiMode )
        // para refrescar la caché, comprobar el run_mode, usar la factory, etc.
        
        if refreshTarget == "all" || refreshTarget == blockName {
            cacheFile := getCacheFilePath(blockName)
            _ = os.Remove(cacheFile) // Ignoramos el error si el archivo no existe
        }

        blockConfig, _ := cfg.Blocks[blockName].(map[string]interface{})    
        runMode, _ := blockConfig["run_mode"].(string)
        if runMode == "" { runMode = "all" }
        // Si el bloque está configurado para ejecutarse solo en modo 'tty', saltamos.
        if runMode == "tty" { continue }

        blockType, _ := blockConfig["type"].(string)
        if factory, ok := blockFactory[blockType]; ok {
            b := factory()
            blockConfig["name"] = blockName
            if err := b.Init(blockConfig, cfg.General, theme); err != nil {
                log.Printf("Error inicializando bloque '%s': %v", blockName, err)
                continue
            }
            activeBlocks = append(activeBlocks, b)
        }
    }

    return &SetupResult{
        Config:       cfg,
        Theme:        theme,
        ActiveBlocks: activeBlocks,
        BlockFactory: blockFactory,
    }, nil
}