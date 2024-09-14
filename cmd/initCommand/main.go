package initCommand

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/config"
	"os"
	"path/filepath"
)

type InitCommand struct {
	CobraCommand *cobra.Command
}

func NewInitCommand() *InitCommand {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize sisho configuration",
		Long:  `Create a new sisho.yml configuration file in the current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			currentDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			configPath := filepath.Join(currentDir, "sisho.yml")

			// Check if sisho.yml already exists
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("sisho.yml already exists in the current directory")
			}

			// Create default configuration
			defaultConfig := config.Config{
				Lang: "ja", // Default language set to Japanese
				AutoCollect: config.AutoCollect{
					ReadmeMd:     true,
					TargetCodeMd: true,
				},
			}

			configHolder := config.ConfigHolder{
				Path:    configPath,
				RootDir: currentDir,
				Config:  defaultConfig,
			}

			// Write configuration file
			err = config.WriteConfig(configHolder)
			if err != nil {
				return fmt.Errorf("failed to write sisho.yml: %w", err)
			}

			fmt.Println("sisho.yml has been created successfully.")
			return nil
		},
	}

	return &InitCommand{
		CobraCommand: cmd,
	}
}
