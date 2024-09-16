package cmd

import (
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/cmd/addCommand"
	"github.com/t-kuni/sisho/cmd/initCommand"
	"github.com/t-kuni/sisho/cmd/makeCommand"
	"github.com/t-kuni/sisho/infrastructure/external/claude"
)

type RootCommand struct {
	CobraCommand *cobra.Command
}

func NewRootCommand() *RootCommand {
	cmd := &cobra.Command{
		Use:   "sisho",
		Short: "A tool for scaffolding using LLM",
		Long:  `Sisho is a command-line tool for scaffolding projects using Large Language Models (LLM).`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	claudeClient := claude.NewClaudeClient()

	cmd.AddCommand(initCommand.NewInitCommand().CobraCommand)
	cmd.AddCommand(makeCommand.NewMakeCommand(claudeClient).CobraCommand)
	cmd.AddCommand(addCommand.NewAddCommand().CobraCommand)

	return &RootCommand{
		CobraCommand: cmd,
	}
}
