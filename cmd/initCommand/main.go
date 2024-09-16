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
		Short: "Initialize a new Sisho project",
		Long:  `Initialize a new Sisho project by creating a sisho.yml configuration file in the current directory.`,
		RunE:  runInit,
	}

	return &InitCommand{
		CobraCommand: cmd,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
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
		Lang: "en",
		AutoCollect: config.AutoCollect{
			ReadmeMd:     true,
			TargetCodeMd: true,
		},
		AdditionalKnowledge: config.AdditionalKnowledge{
			FolderStructure: true,
		},
	}

	// Write configuration to sisho.yml
	err = config.WriteConfig(config.ConfigHolder{
		Path:   configPath,
		Config: defaultConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to write sisho.yml: %w", err)
	}

	fmt.Println("Initialized Sisho project. Created sisho.yml with default configuration.")
	return nil
}
