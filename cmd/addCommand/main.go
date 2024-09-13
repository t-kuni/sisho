package addCommand

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/t-kuni/llm-coding-example/sisho/kinds"
	"github.com/t-kuni/llm-coding-example/sisho/knowledge"
	"os"
	"path/filepath"
)

type AddCommand struct {
	CobraCommand *cobra.Command
}

func NewAddCommand() *AddCommand {
	cmd := &cobra.Command{
		Use:   "add [kind] [path]",
		Short: "Add a file to .knowledges.yml",
		Long:  `Add a specified file to .knowledges.yml in the current directory. If .knowledges.yml doesn't exist, it will be created.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kindName := kinds.KindName(args[0])
			path := args[1]

			// Validate kind
			if _, ok := kinds.GetKind(kindName); !ok {
				return fmt.Errorf("invalid kind: %s", kindName)
			}

			// Check if file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", path)
			}

			// Read existing knowledge or create new
			f, err := knowledge.ReadKnowledge(".knowledge.yml")
			if err != nil && !os.IsNotExist(err) {
				return err
			}

			knowList := f.KnowledgeList

			// Add new knowledge
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			newKnowledge := knowledge.Knowledge{
				Path: absPath,
				Kind: string(kindName),
			}
			knowList = append(knowList, newKnowledge)

			// Write updated knowledge
			err = knowledge.WriteKnowledge(".knowledge.yml", knowledge.KnowledgeFile{KnowledgeList: knowList})
			if err != nil {
				return err
			}

			fmt.Printf("Added %s to .knowledge.yml with kind %s\n", path, kindName)
			return nil
		},
	}

	return &AddCommand{
		CobraCommand: cmd,
	}
}
