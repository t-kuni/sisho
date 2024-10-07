package addCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/model/kinds"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"os"
	"path/filepath"
)

type AddCommand struct {
	CobraCommand *cobra.Command
}

func NewAddCommand(knowledgeRepo knowledge.Repository) *AddCommand {
	cmd := &cobra.Command{
		Use:   "add [kind] [path]",
		Short: "Add a file to .knowledge.yml",
		Long:  `Add a specified file to .knowledge.yml in the current directory. If .knowledge.yml doesn't exist, it will be created.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kindName := kinds.KindName(args[0])
			path := args[1]

			// Validate kind
			if _, ok := kinds.GetKind(kindName); !ok {
				return eris.Errorf("invalid kind: %s", kindName)
			}

			// Check if file exists
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return eris.Errorf("file not found: %s", path)
			}

			// Read existing knowledge or create new
			knowledgeFile, err := knowledgeRepo.Read(".knowledge.yml")
			if err != nil && !os.IsNotExist(err) {
				return eris.Wrap(err, "failed to read .knowledge.yml")
			}

			knowList := knowledgeFile.KnowledgeList

			// Add new knowledge
			relPath, err := filepath.Rel(".", path)
			if err != nil {
				return eris.Wrap(err, "failed to get relative path")
			}
			newKnowledge := knowledge.Knowledge{
				Path: relPath,
				Kind: kindName,
			}
			knowList = append(knowList, newKnowledge)

			// Write updated knowledge
			err = knowledgeRepo.Write(".knowledge.yml", knowledge.KnowledgeFile{KnowledgeList: knowList})
			if err != nil {
				return eris.Wrap(err, "failed to write .knowledge.yml")
			}

			fmt.Printf("Added %s to .knowledge.yml with kind %s\n", relPath, kindName)
			return nil
		},
	}

	return &AddCommand{
		CobraCommand: cmd,
	}
}