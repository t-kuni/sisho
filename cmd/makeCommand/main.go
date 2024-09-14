package makeCommand

import (
	"errors"
	"fmt"
	"github.com/segmentio/ksuid"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/chat/claude"
	"github.com/t-kuni/sisho/config"
	"github.com/t-kuni/sisho/knowledge"
	"github.com/t-kuni/sisho/prompts"
	"github.com/t-kuni/sisho/prompts/oneMoreMake"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type MakeCommand struct {
	CobraCommand *cobra.Command
}

func NewMakeCommand() *MakeCommand {
	var promptFlag bool

	cmd := &cobra.Command{
		Use:   "make [path...]",
		Short: "Generate files using LLM",
		Long:  `Generate files at the specified paths using LLM based on the knowledge sets.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runMake(&promptFlag),
	}

	cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Open editor for additional instructions")

	return &MakeCommand{
		CobraCommand: cmd,
	}
}

func runMake(promptFlag *bool) func(cmd *cobra.Command, args []string) error {
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
				prompt, err = prompts.BuildPrompt(prompts.PromptParam{
					KnowledgeSets: knowledgeSets,
					Targets:       []prompts.Target{target},
					Instructions:  instructions,
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

			fmt.Println(answer)
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
