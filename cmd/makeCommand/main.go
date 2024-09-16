package makeCommand

import (
	"errors"
	"fmt"
	"github.com/segmentio/ksuid"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/chat/claude"
	"github.com/t-kuni/sisho/config"
	"github.com/t-kuni/sisho/knowledge"
	"github.com/t-kuni/sisho/prompts"
	"github.com/t-kuni/sisho/prompts/oneMoreMake"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type MakeCommand struct {
	CobraCommand *cobra.Command
}

func NewMakeCommand() *MakeCommand {
	var promptFlag bool
	var applyFlag bool

	cmd := &cobra.Command{
		Use:   "make [path...]",
		Short: "Generate files using LLM",
		Long:  `Generate files at the specified paths using LLM based on the knowledge sets.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runMake(&promptFlag, &applyFlag),
	}

	cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Open editor for additional instructions")
	cmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Apply LLM output to files")

	return &MakeCommand{
		CobraCommand: cmd,
	}
}

func runMake(promptFlag *bool, applyFlag *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		configHolder, err := config.ReadConfig()
		if err != nil {
			return err
		}

		scannedKnowledge, err := knowledge.ScanKnowledge(configHolder.RootDir, args, configHolder.Config)
		if err != nil {
			return err
		}

		knowledgeSets, err := knowledge.ConvertToKnowledgeSet(configHolder.RootDir, scannedKnowledge)
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

		chat := claude.ClaudeChat{}

		for i, path := range args {
			target, err := readTarget(path)
			if err != nil {
				return err
			}

			var prompt string
			if i == 0 {
				folderStructure := ""
				if configHolder.Config.AdditionalKnowledge.FolderStructure {
					folderStructure, err = getFolderStructure(configHolder.RootDir)
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

			answer, err := chat.Send(prompt)
			if err != nil {
				return err
			}

			err = saveHistory(configHolder.RootDir, prompt, answer)
			if err != nil {
				return err
			}

			if *applyFlag {
				err = applyChanges(path, answer)
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

func readTarget(path string) (prompts.Target, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return prompts.Target{
				Path:    path,
				Content: "",
			}, nil
		}
		return prompts.Target{}, err
	}

	return prompts.Target{
		Path:    path,
		Content: string(content),
	}, nil
}

func getAdditionalInstructions() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tempFile, err := os.CreateTemp("", "sisho-instructions-*.txt")
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

	return strings.TrimSpace(string(instructions)), nil
}

func saveHistory(rootDir, prompt, answer string) error {
	historyDir := filepath.Join(rootDir, ".sisho", "history")
	err := os.MkdirAll(historyDir, 0755)
	if err != nil {
		return err
	}

	id := ksuid.New()
	singleHistoryDir := filepath.Join(historyDir, id.String())
	err = os.Mkdir(singleHistoryDir, 0755)
	if err != nil {
		return err
	}

	timestampFile := filepath.Join(singleHistoryDir, time.Now().Format("2006-01-02T15:04:05"))
	_, err = os.Create(timestampFile)
	if err != nil {
		return err
	}

	promptFile := filepath.Join(singleHistoryDir, "prompt.md")
	err = os.WriteFile(promptFile, []byte(prompt), 0644)
	if err != nil {
		return err
	}

	answerFile := filepath.Join(singleHistoryDir, "answer.md")
	err = os.WriteFile(answerFile, []byte(answer), 0644)
	if err != nil {
		return err
	}

	return nil
}

func printKnowledgePaths(knowledgeSets []prompts.KnowledgeSet) {
	fmt.Println("Knowledge paths used:")
	for _, set := range knowledgeSets {
		for _, k := range set.Knowledge {
			fmt.Printf("- %s\n", k.Path)
		}
	}
	fmt.Println("")
}

func applyChanges(path string, answer string) error {
	re := regexp.MustCompile("(?s)(\n|^)<!-- CODE_BLOCK_BEGIN -->```" + regexp.QuoteMeta(path) + "(\n|^)(.*?)```<!-- CODE_BLOCK_END -->(\n|$)")
	matches := re.FindStringSubmatch(answer)

	if len(matches) < 5 {
		return fmt.Errorf("no code block found for %s", path)
	}

	newContent := strings.TrimSpace(matches[3])

	oldContent, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if string(oldContent) != newContent {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(oldContent), newContent, false)
		fmt.Printf("Changes for %s:\n", path)
		fmt.Println(dmp.DiffPrettyText(diffs))

		err = os.WriteFile(path, []byte(newContent), 0644)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("No changes needed for %s\n", path)
	}

	return nil
}

func getFolderStructure(rootDir string) (string, error) {
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
