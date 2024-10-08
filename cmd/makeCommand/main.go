package makeCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/service/make"
	"io"
	"os"
	"os/exec"
	"strings"
)

// MakeCommand は、makeコマンドの構造体です。
type MakeCommand struct {
	CobraCommand *cobra.Command
}

// NewMakeCommand は、MakeCommandの新しいインスタンスを作成します。
func NewMakeCommand(makeService *make.MakeService) *MakeCommand {
	var promptFlag bool
	var applyFlag bool
	var chainFlag bool
	var inputFlag bool

	cmd := &cobra.Command{
		Use:   "make [path...]",
		Short: "Generate files using LLM",
		Long:  `Generate files at the specified paths using LLM based on the knowledge sets.`,
		Args:  cobra.MinimumNArgs(1),
		RunE:  runMake(&promptFlag, &applyFlag, &chainFlag, &inputFlag, makeService),
	}

	cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Open editor for additional instructions")
	cmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Apply LLM output to files")
	cmd.Flags().BoolVarP(&chainFlag, "chain", "c", false, "Include dependent files based on deps-graph")
	cmd.Flags().BoolVarP(&inputFlag, "input", "i", false, "Read additional instructions from stdin")

	return &MakeCommand{
		CobraCommand: cmd,
	}
}

// runMake は、makeコマンドの主要なロジックを実行します。
func runMake(
	promptFlag *bool,
	applyFlag *bool,
	chainFlag *bool,
	inputFlag *bool,
	makeService *make.MakeService,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// 追加の指示の取得
		var instructions string
		if *promptFlag && *inputFlag {
			return eris.New("cannot use both -p and -i flags")
		} else if *promptFlag {
			var err error
			instructions, err = getAdditionalInstructions()
			if err != nil {
				return eris.Wrap(err, "failed to get additional instructions")
			}
			fmt.Println("Additional instructions:")
			fmt.Println(instructions)
		} else if *inputFlag {
			var err error
			instructions, err = readStdin()
			if err != nil {
				return eris.Wrap(err, "failed to read from stdin")
			}
			fmt.Println("Additional instructions:")
			fmt.Println(instructions)
		}

		err := makeService.Make(args, *applyFlag, *chainFlag, instructions)
		if err != nil {
			return eris.Wrap(err, "failed to execute make command")
		}

		return nil
	}
}

// getAdditionalInstructions は、ユーザーから追加の指示を取得します。
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

// readStdin は標準入力からテキストを読み取ります。
func readStdin() (string, error) {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", eris.Wrap(err, "failed to read from stdin")
	}
	return strings.TrimSpace(string(stdin)), nil
}
