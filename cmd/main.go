package cmd

import (
	"github.com/spf13/cobra"
	"github.com/t-kuni/llm-coding-example/sisho/cmd/addCommand"
	"github.com/t-kuni/llm-coding-example/sisho/cmd/initCommand"
	"github.com/t-kuni/llm-coding-example/sisho/cmd/makeCommand"
)

type RootCommand struct {
	CobraCommand *cobra.Command
}

func NewRootCommand() *RootCommand {
	cmd := &cobra.Command{
		Use:   "",
		Short: "",
		Long:  ``,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(initCommand.NewInitCommand().CobraCommand)
	cmd.AddCommand(makeCommand.NewMakeCommand().CobraCommand)
	cmd.AddCommand(addCommand.NewAddCommand().CobraCommand)

	return &RootCommand{
		CobraCommand: cmd,
	}
}
