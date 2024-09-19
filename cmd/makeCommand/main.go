package makeCommand

import (
	"errors"
	"fmt"
	"github.com/segmentio/ksuid"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	chat2 "github.com/t-kuni/sisho/domain/model/chat"
	modelClaude "github.com/t-kuni/sisho/domain/model/chat/claude"
	modelOpenAi "github.com/t-kuni/sisho/domain/model/chat/openAi"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"github.com/t-kuni/sisho/domain/model/prompts/oneMoreMake"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type MakeCommand struct {
	CobraCommand *cobra.Command
	claudeClient claude.Client
	openAiClient openAi.Client
}

func NewMakeCommand(
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	autoCollectService *autoCollect.AutoCollectService,
	contextScanService *contextScan.ContextScanService,
	knowledgeScanService *knowledgeScan.KnowledgeScanService,
	knowledgeLoadService *knowledgeLoad.KnowledgeLoadService,
) *MakeCommand {
	var promptFlag bool
	var applyFlag bool

	cmd := &cobra.Command{
		Use:   "make [path...]",
		Short: "Generate files using LLM",
		Long:  `Generate files at the specified paths using LLM based on the knowledge sets.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: runMake(&promptFlag, &applyFlag, claudeClient, openAiClient, configFindService, configRepository,
			fileRepository, autoCollectService, contextScanService, knowledgeScanService, knowledgeLoadService),
	}

	cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Open editor for additional instructions")
	cmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Apply LLM output to files")

	return &MakeCommand{
		CobraCommand: cmd,
		claudeClient: claudeClient,
		openAiClient: openAiClient,
	}
}

func runMake(
	promptFlag *bool,
	applyFlag *bool,
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	autoCollectService *autoCollect.AutoCollectService,
	contextScanService *contextScan.ContextScanService,
	knowledgeScanService *knowledgeScan.KnowledgeScanService,
	knowledgeLoadService *knowledgeLoad.KnowledgeLoadService,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		configPath, err := configFindService.FindConfig()
		if err != nil {
			return err
		}

		cfg, err := configRepository.Read(configPath)
		if err != nil {
			return err
		}

		rootDir := configFindService.GetProjectRoot(configPath)

		scannedKnowledge, err := knowledgeScanService.ScanKnowledge(rootDir, args)
		if err != nil {
			return err
		}

		knowledgeSets, err := knowledgeLoadService.LoadKnowledge(rootDir, scannedKnowledge)
		if err != nil {
			return err
		}

		var instructions string
		if *promptFlag {
			instructions, err = getAdditionalInstructions()
			if err != nil {
				return err
			}
		}

		printKnowledgePaths(knowledgeSets)

		var chat chat2.Chat

		switch cfg.LLM.Driver {
		case "open-ai":
			chat = modelOpenAi.NewOpenAiChat(openAiClient)
		case "anthropic":
			chat = modelClaude.NewClaudeChat(claudeClient)
		default:
			return fmt.Errorf("unsupported LLM driver: %s", cfg.LLM.Driver)
		}

		fmt.Printf("Using LLM: %s with model: %s\n", cfg.LLM.Driver, cfg.LLM.Model)

		historyDir, err := createHistoryDir(rootDir)
		if err != nil {
			return err
		}

		for i, path := range args {
			target, err := readTarget(path, fileRepository)
			if err != nil {
				return err
			}

			var prompt string
			if i == 0 {
				folderStructure := ""
				if cfg.AdditionalKnowledge.FolderStructure {
					folderStructure, err = getFolderStructure(rootDir, fileRepository)
					if err != nil {
						return err
					}
				}

				prompt, err = prompts.BuildPrompt(prompts.PromptParam{
					KnowledgeSets:   knowledgeSets,
					Targets:         []prompts.Target{target},
					Instructions:    instructions,
					FolderStructure: folderStructure,
				})
			} else {
				prompt, err = oneMoreMake.BuildPrompt(oneMoreMake.PromptParam{
					Path: path,
				})
			}
			if err != nil {
				return err
			}

			answer, err := chat.Send(prompt, cfg.LLM.Model)
			if err != nil {
				return err
			}

			err = saveHistory(historyDir, prompt, answer)
			if err != nil {
				return err
			}

			if *applyFlag {
				err = applyChanges(path, answer, fileRepository)
				if err != nil {
					return err
				}
				fmt.Printf("Applied changes to %s\n", path)
			} else {
				fmt.Println(answer)
			}
		}

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
		return "", err
	}
	defer os.Remove(tempFile.Name())

	cmd := exec.Command(editor, tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", err
	}

	instructions, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return "", err
	}

	return string(instructions), nil
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

func readTarget(path string, fileRepository file.Repository) (prompts.Target, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return prompts.Target{}, err
		}
		content = []byte{}
	}

	return prompts.Target{
		Path:    path,
		Content: string(content),
	}, nil
}

func createHistoryDir(rootDir string) (string, error) {
	historyBaseDir := filepath.Join(rootDir, ".sisho", "history")
	err := os.MkdirAll(historyBaseDir, 0755)
	if err != nil {
		return "", err
	}

	id := ksuid.New()
	historyDir := filepath.Join(historyBaseDir, id.String())
	err = os.Mkdir(historyDir, 0755)
	if err != nil {
		return "", err
	}

	timeFile := filepath.Join(historyDir, time.Now().Format("2006-01-02T15:04:05"))
	_, err = os.Create(timeFile)
	if err != nil {
		return "", err
	}

	return historyDir, nil
}

func saveHistory(historyDir, prompt, answer string) error {
	err := os.WriteFile(filepath.Join(historyDir, "prompt.md"), []byte(prompt), 0644)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(historyDir, "answer.md"), []byte(answer), 0644)
}

func applyChanges(path, answer string, fileRepository file.Repository) error {
	// この正規表現は絶対に変更しないでください
	// gpt-4だとコードブロックの終了からコメントの間に1文字入ることがある
	re := regexp.MustCompile("(?s)(\n|^)<!-- CODE_BLOCK_BEGIN -->```" + regexp.QuoteMeta(path) + "(.*)```.?<!-- CODE_BLOCK_END -->(\n|$)")
	matches := re.FindStringSubmatch(answer)

	if len(matches) < 4 {
		return errors.New("no code block found in the answer")
	}

	newContent := strings.TrimSpace(matches[2])

	oldContent, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if string(oldContent) != newContent {
		err = write(path, []byte(newContent))
		if err != nil {
			return err
		}

		printDiff(string(oldContent), newContent)
	}

	return nil
}

func printDiff(oldContent, newContent string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)
	fmt.Println(dmp.DiffPrettyText(diffs))
}

func getFolderStructure(rootDir string, fileRepository file.Repository) (string, error) {
	var structure strings.Builder
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		indent := strings.Repeat("  ", strings.Count(relPath, string(os.PathSeparator)))
		if info.IsDir() {
			structure.WriteString(fmt.Sprintf("%s/%s\n", indent, info.Name()))
		} else {
			structure.WriteString(fmt.Sprintf("%s%s\n", indent, info.Name()))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return structure.String(), nil
}

func write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
