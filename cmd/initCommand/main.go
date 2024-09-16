package initCommand

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/repository/config"
	"os"
	"path/filepath"
)

type InitCommand struct {
	CobraCommand *cobra.Command
}

func NewInitCommand(configRepository config.Repository) *InitCommand {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Sisho project",
		Long:  `Initialize a new Sisho project by creating a sisho.yml configuration file in the current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			currentDir, err := os.Getwd()
			if err != nil {
				return err
			}

			configPath := filepath.Join(currentDir, "sisho.yml")
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("sisho.yml already exists in the current directory")
			}

			cfg := &config.Config{
				Lang: "en",
				AutoCollect: config.AutoCollect{
					ReadmeMd:     true,
					TargetCodeMd: true,
				},
				AdditionalKnowledge: config.AdditionalKnowledge{
					FolderStructure: true,
				},
			}

			err = configRepository.Write(configPath, cfg)
			if err != nil {
				return err
			}

			fmt.Println("Initialized Sisho project. Created sisho.yml in the current directory.")
			return nil
		},
	}

	return &InitCommand{
		CobraCommand: cmd,
	}
}
