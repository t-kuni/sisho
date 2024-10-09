package qCommand

import (
	"fmt"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"github.com/t-kuni/sisho/domain/model/prompts/question"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/model/chat"
	modelClaude "github.com/t-kuni/sisho/domain/model/chat/claude"
	modelOpenAi "github.com/t-kuni/sisho/domain/model/chat/openAi"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
)

type QCommand struct {
	CobraCommand *cobra.Command
}

func NewQCommand(
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	knowledgeScanService *knowledgeScan.KnowledgeScanService,
	knowledgeLoadService *knowledgeLoad.KnowledgeLoadService,
	timer timer.ITimer,
	ksuidGenerator ksuid.IKsuid,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
) *QCommand {
	var promptFlag bool
	var inputFlag bool

	cmd := &cobra.Command{
		Use:   "q [path...]",
		Short: "Ask questions about specified files using LLM",
		Long:  `Ask questions about specified files using LLM based on the knowledge sets.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: runQ(&promptFlag, &inputFlag, claudeClient, openAiClient, configFindService, configRepository,
			fileRepository, knowledgeScanService, knowledgeLoadService, timer, ksuidGenerator, folderStructureMakeService),
	}

	cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Open editor for additional instructions")
	cmd.Flags().BoolVarP(&inputFlag, "input", "i", false, "Read additional instructions from stdin")

	return &QCommand{
		CobraCommand: cmd,
	}
}

func runQ(
	promptFlag *bool,
	inputFlag *bool,
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	knowledgeScanService *knowledgeScan.KnowledgeScanService,
	knowledgeLoadService *knowledgeLoad.KnowledgeLoadService,
	timer timer.ITimer,
	ksuidGenerator ksuid.IKsuid,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		configPath, err := configFindService.FindConfig()
		if err != nil {
			return eris.Wrap(err, "failed to find config file")
		}

		cfg, err := configRepository.Read(configPath)
		if err != nil {
			return eris.Wrap(err, "failed to read config file")
		}

		rootDir := configFindService.GetProjectRoot(configPath)

		fmt.Println("Target Codes:")
		for _, arg := range args {
			fmt.Printf("- %s\n", arg)
		}
		fmt.Println()

		var instructions string
		if *promptFlag && *inputFlag {
			return eris.New("cannot use both -p and -i flags")
		} else if *promptFlag {
			instructions, err = getAdditionalInstructions()
			if err != nil {
				return eris.Wrap(err, "failed to get additional instructions")
			}
			fmt.Println("Additional instructions:")
			fmt.Println(instructions)
		} else if *inputFlag {
			instructions, err = readStdin()
			if err != nil {
				return eris.Wrap(err, "failed to read from stdin")
			}
			fmt.Println("Additional instructions:")
			fmt.Println(instructions)
		}

		fmt.Printf("Using LLM: %s with model: %s\n", cfg.LLM.Driver, cfg.LLM.Model)

		historyDir, err := createHistoryDir(rootDir, timer, ksuidGenerator)
		if err != nil {
			return eris.Wrap(err, "failed to create history directory")
		}

		var folderStructure string
		if cfg.AdditionalKnowledge.FolderStructure {
			folderStructure, err = folderStructureMakeService.MakeTree(rootDir)
			if err != nil {
				return eris.Wrap(err, "failed to get folder structure")
			}
		}

		var chat chat.Chat
		switch cfg.LLM.Driver {
		case "open-ai":
			chat = modelOpenAi.NewOpenAiChat(openAiClient)
		case "anthropic":
			chat = modelClaude.NewClaudeChat(claudeClient)
		default:
			return eris.Errorf("unsupported LLM driver: %s", cfg.LLM.Driver)
		}

		targets, err := readAllTargets(args, fileRepository)
		if err != nil {
			return eris.Wrap(err, "failed to read all targets")
		}

		scannedKnowledge, err := knowledgeScanService.ScanKnowledgeMultipleTarget(rootDir, args)
		if err != nil {
			return eris.Wrap(err, "failed to scan knowledge")
		}

		knowledgeSets, err := knowledgeLoadService.LoadKnowledge(rootDir, scannedKnowledge)
		if err != nil {
			return eris.Wrap(err, "failed to load knowledge")
		}

		printKnowledgePaths(knowledgeSets)

		prompt, err := question.BuildPrompt(question.PromptParam{
			KnowledgeSets:   knowledgeSets,
			Targets:         targets,
			Question:        instructions,
			FolderStructure: folderStructure,
		})
		if err != nil {
			return eris.Wrap(err, "failed to build prompt")
		}

		err = savePromptHistory(historyDir, prompt)
		if err != nil {
			return eris.Wrap(err, "failed to save prompt history")
		}

		answer, err := chat.Send(prompt, cfg.LLM.Model)
		if err != nil {
			return eris.Wrap(err, "failed to send message to LLM")
		}

		err = saveAnswerHistory(historyDir, answer.Content)
		if err != nil {
			return eris.Wrap(err, "failed to save answer history")
		}

		fmt.Println(answer.Content)

		return nil
	}
}

func getAdditionalInstructions() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tempFile, err := os.CreateTemp("", "sisho-instructions-*.md")
	if err != nil {
		return "", eris.Wrap(err, "failed to create temporary file")
	}
	defer os.Remove(tempFile.Name())

	cmd := exec.Command(editor, tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", eris.Wrap(err, "failed to run editor")
	}

	instructions, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return "", eris.Wrap(err, "failed to read instructions from temporary file")
	}

	return strings.TrimSpace(string(instructions)), nil
}

func readStdin() (string, error) {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", eris.Wrap(err, "failed to read from stdin")
	}
	return strings.TrimSpace(string(stdin)), nil
}

func printKnowledgePaths(knowledgeSets []prompts.KnowledgeSet) {
	fmt.Println("Knowledge paths:")
	for _, set := range knowledgeSets {
		for _, k := range set.Knowledge {
			fmt.Printf("- %s (%s)\n", k.Path, set.Kind)
		}
	}
	fmt.Println()
}

func readAllTargets(paths []string, fileRepository file.Repository) ([]prompts.Target, error) {
	targets := make([]prompts.Target, len(paths))
	for i, path := range paths {
		target, err := readTarget(path, fileRepository)
		if err != nil {
			return nil, eris.Wrapf(err, "failed to read target: %s", path)
		}
		targets[i] = target
	}
	return targets, nil
}

func readTarget(path string, fileRepository file.Repository) (prompts.Target, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return prompts.Target{}, eris.Wrapf(err, "failed to read file: %s", path)
		}
		content = []byte{}
	}

	return prompts.Target{
		Path:    path,
		Content: string(content),
	}, nil
}

func createHistoryDir(rootDir string, timer timer.ITimer, ksuidGenerator ksuid.IKsuid) (string, error) {
	historyBaseDir := filepath.Join(rootDir, ".sisho", "history", "questions")
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

func savePromptHistory(historyDir string, prompt string) error {
	filename := "prompt.md"
	err := os.WriteFile(filepath.Join(historyDir, filename), []byte(prompt), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write prompt to history")
	}
	return nil
}

func saveAnswerHistory(historyDir string, answer string) error {
	filename := "answer.md"
	err := os.WriteFile(filepath.Join(historyDir, filename), []byte(answer), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write answer to history")
	}
	return nil
}
