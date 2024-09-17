package cmd

import (
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/cmd/addCommand"
	"github.com/t-kuni/sisho/cmd/initCommand"
	"github.com/t-kuni/sisho/cmd/makeCommand"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/infrastructure/external/claude"
	"github.com/t-kuni/sisho/infrastructure/external/openAi"
	configRepo "github.com/t-kuni/sisho/infrastructure/repository/config"
	fileRepo "github.com/t-kuni/sisho/infrastructure/repository/file"
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
	openAiClient := openAi.NewOpenAIClient()
	fileRepository := fileRepo.NewFileRepository()
	configRepository := configRepo.NewConfigRepository()
	configFindSrv := configFindService.NewConfigFindService(fileRepository)
	contextScanSrv := contextScan.NewContextScanService(fileRepository)
	autoCollectSrv := autoCollect.NewAutoCollectService(configRepository, fileRepository, contextScanSrv)

	cmd.AddCommand(initCommand.NewInitCommand(configRepository).CobraCommand)
	cmd.AddCommand(makeCommand.NewMakeCommand(
		claudeClient,
		openAiClient,
		configFindSrv,
		configRepository,
		fileRepository,
		autoCollectSrv,
		contextScanSrv,
	).CobraCommand)
	cmd.AddCommand(addCommand.NewAddCommand().CobraCommand)

	return &RootCommand{
		CobraCommand: cmd,
	}
}
