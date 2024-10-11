package fixTaskCommand

import (
	"encoding/json"
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/model/chat"
	"github.com/t-kuni/sisho/domain/model/prompts/extractPaths"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/service/chatFactory"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/extractCodeBlock"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/make"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FixTaskCommand struct {
	CobraCommand *cobra.Command
}

func NewFixTaskCommand(
	configFindService *configFindService.ConfigFindService,
	configRepo config.Repository,
	makeService *make.MakeService,
	chatFactory *chatFactory.ChatFactory,
	timer timer.ITimer,
	ksuidGenerator ksuid.IKsuid,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
	extractCodeBlockService *extractCodeBlock.CodeBlockExtractService,
) *FixTaskCommand {
	var tryCount int

	cmd := &cobra.Command{
		Use:   "fix:task [taskName]",
		Short: "Run a task and fix errors using LLM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskName := args[0]

			cfg, projectRoot, err := loadConfig(configFindService, configRepo)
			if err != nil {
				return err
			}

			task, err := findTask(cfg, taskName)
			if err != nil {
				return err
			}

			historyDir, err := createHistoryDir(projectRoot, timer, ksuidGenerator)
			if err != nil {
				return err
			}

			chat, err := chatFactory.Make(cfg)
			if err != nil {
				return eris.Wrap(err, "failed to create chat model")
			}

			for i := 0; i < tryCount; i++ {
				fmt.Printf("Attempt %d/%d\n", i+1, tryCount)

				stdout, stderr, err := runTask(task, projectRoot)
				if err == nil {
					fmt.Println("Task completed successfully")
					return nil
				}

				errorMessage := buildErrorMessage(stdout, stderr, err)

				paths, err := getPathsToFix(chat, cfg, task.Run, errorMessage, historyDir, i+1, projectRoot, folderStructureMakeService, extractCodeBlockService)
				if err != nil {
					return err
				}

				if len(paths) == 0 {
					fmt.Println("No files to fix. Stopping the process.")
					return nil
				}

				fmt.Println("Files to fix:")
				for _, path := range paths {
					fmt.Printf("- %s\n", path)
				}

				err = makeService.Make(paths, true, false, errorMessage)
				if err != nil {
					return eris.Wrap(err, "failed to fix files")
				}
			}

			// Run the task one last time to check if it's fixed
			stdout, stderr, err := runTask(task, projectRoot)
			if err == nil {
				fmt.Println("Task completed successfully after fixes")
				return nil
			}

			errorMessage := buildErrorMessage(stdout, stderr, err)
			return eris.New(fmt.Sprintf("failed to fix the task after maximum attempts. Last error: %s", errorMessage))
		},
	}

	cmd.Flags().IntVarP(&tryCount, "try", "t", 1, "Number of attempts to fix the task")

	return &FixTaskCommand{
		CobraCommand: cmd,
	}
}

// loadConfig loads the configuration file and returns the config and project root
func loadConfig(configFindService *configFindService.ConfigFindService, configRepo config.Repository) (*config.Config, string, error) {
	configPath, err := configFindService.FindConfig()
	if err != nil {
		return nil, "", eris.Wrap(err, "failed to find config file")
	}

	cfg, err := configRepo.Read(configPath)
	if err != nil {
		return nil, "", eris.Wrap(err, "failed to read config file")
	}

	projectRoot := configFindService.GetProjectRoot(configPath)
	return cfg, projectRoot, nil
}

// findTask finds the specified task in the configuration
func findTask(cfg *config.Config, taskName string) (*config.Task, error) {
	for _, t := range cfg.Tasks {
		if t.Name == taskName {
			return &t, nil
		}
	}
	return nil, eris.Errorf("task not found: %s", taskName)
}

// runTask executes the specified task and returns stdout, stderr, and error
func runTask(task *config.Task, projectRoot string) (string, string, error) {
	cmd := exec.Command("sh", "-c", task.Run)
	cmd.Dir = projectRoot

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// buildErrorMessage constructs an error message from stdout, stderr, and error
func buildErrorMessage(stdout, stderr string, err error) string {
	return fmt.Sprintf("Stdout:\n%s\nStderr:\n%s\nError:\n%s", stdout, stderr, err.Error())
}

// getPathsToFix gets the paths that need to be fixed based on the error message
func getPathsToFix(
	chat chat.Chat,
	cfg *config.Config,
	command string,
	errorMessage string,
	historyDir string,
	attempt int,
	projectRoot string,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
	extractCodeBlockService *extractCodeBlock.CodeBlockExtractService,
) ([]string, error) {
	var folderStructure string
	var err error

	if cfg.AdditionalKnowledge.FolderStructure {
		folderStructure, err = folderStructureMakeService.MakeTree(projectRoot)
		if err != nil {
			return nil, eris.Wrap(err, "failed to create folder structure")
		}
	}

	prompt, err := extractPaths.BuildPrompt(extractPaths.PromptParam{
		Commands:        command,
		CommandResult:   errorMessage,
		FolderStructure: folderStructure,
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to build prompt")
	}

	err = savePromptHistory(historyDir, attempt, prompt)
	if err != nil {
		return nil, err
	}

	result, err := chat.Send(prompt, cfg.LLM.Model)
	if err != nil {
		return nil, eris.Wrap(err, "failed to send message to LLM")
	}
	err = saveAnswerHistory(historyDir, attempt, result.Content)
	if err != nil {
		return nil, err
	}

	jsonContent, err := extractCodeBlockService.ExtractCodeBlock(result.Content, "json")
	if err != nil {
		return nil, eris.Wrap(err, "failed to extract JSON from LLM response")
	}

	var paths extractPaths.ExtractPathsResult
	err = json.Unmarshal([]byte(jsonContent), &paths)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal LLM response")
	}

	// Check if the paths to fix exist
	var validPaths []string
	for _, path := range paths {
		fullPath := filepath.Join(projectRoot, path)
		if _, err := os.Stat(fullPath); err == nil {
			validPaths = append(validPaths, path)
		} else {
			fmt.Printf("Warning: Path does not exist: %s\n", path)
		}
	}

	if len(validPaths) == 0 {
		return nil, eris.New("no valid paths to fix")
	}

	return validPaths, nil
}

// createHistoryDir creates a directory for storing the history of fix attempts
func createHistoryDir(projectRoot string, timer timer.ITimer, ksuidGenerator ksuid.IKsuid) (string, error) {
	historyBaseDir := filepath.Join(projectRoot, ".sisho", "fixTask")
	err := os.MkdirAll(historyBaseDir, 0755)
	if err != nil {
		return "", eris.Wrap(err, "failed to create history base directory")
	}

	id := ksuidGenerator.New()
	historyDir := filepath.Join(historyBaseDir, id)
	err = os.Mkdir(historyDir, 0755)
	if err != nil {
		return "", eris.Wrap(err, "failed to create history directory")
	}

	timeFile := filepath.Join(historyDir, timer.Now().Format("2006-01-02T15:04:05"))
	_, err = os.Create(timeFile)
	if err != nil {
		return "", eris.Wrap(err, "failed to create time file")
	}

	return historyDir, nil
}

// savePromptHistory saves the prompt used for fixing to the history directory
func savePromptHistory(historyDir string, index int, prompt string) error {
	filename := fmt.Sprintf("prompt_%02d.md", index)
	err := os.WriteFile(filepath.Join(historyDir, filename), []byte(prompt), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write prompt to history")
	}
	return nil
}

// saveAnswerHistory saves the LLM's answer to the history directory
func saveAnswerHistory(historyDir string, index int, answer string) error {
	filename := fmt.Sprintf("answer_%02d.md", index)
	err := os.WriteFile(filepath.Join(historyDir, filename), []byte(answer), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write answer to history")
	}
	return nil
}
