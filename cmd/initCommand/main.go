package initCommand

import (
	"github.com/spf13/cobra"
)

type InitCommand struct {
	CobraCommand *cobra.Command
}

func NewInitCommand() *InitCommand {
	// コマンド定義
	cmd := &cobra.Command{
		Use:   "init",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	return &InitCommand{
		CobraCommand: cmd,
	}
}
