package initCommand

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/file"
	"os"
	"path/filepath"
	"strings"
)

type InitCommand struct {
	CobraCommand *cobra.Command
}

func NewInitCommand(configRepository config.Repository, fileRepository file.Repository) *InitCommand {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new Sisho project",
		Long:  `Initialize a new Sisho project by creating a sisho.yml configuration file in the current directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			currentDir, err := fileRepository.Getwd()
			if err != nil {
				return err
			}

			configPath := filepath.Join(currentDir, "sisho.yml")
			if _, err := os.Stat(configPath); err == nil {
				return fmt.Errorf("sisho.yml already exists in the current directory")
			}

			cfg := &config.Config{
				Lang: "en",
				LLM: config.LLM{
					Driver: "anthropic",
					Model:  "claude-3-5-sonnet-20240620",
				},
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

			// Create .sisho/history folder
			historyDir := filepath.Join(currentDir, ".sisho", "history")
			err = os.MkdirAll(historyDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create .sisho/history folder: %v", err)
			}

			// Update .gitignore
			gitignorePath := filepath.Join(currentDir, ".gitignore")
			gitignoreContent, err := os.ReadFile(gitignorePath)
			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to read .gitignore: %v", err)
			}

			if !contains(string(gitignoreContent), "/.sisho") {
				newContent := string(gitignoreContent)
				if len(newContent) > 0 && !strings.HasSuffix(newContent, "\n") {
					newContent += "\n"
				}
				newContent += "/.sisho\n"
				err = write(gitignorePath, []byte(newContent))
				if err != nil {
					return fmt.Errorf("failed to update .gitignore: %v", err)
				}
			}

			fmt.Println("Initialized Sisho project:")
			fmt.Println("- Created sisho.yml in the current directory")
			fmt.Println("- Created .sisho/history folder")
			fmt.Println("- Updated .gitignore to ignore /.sisho")
			return nil
		},
	}

	return &InitCommand{
		CobraCommand: cmd,
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
