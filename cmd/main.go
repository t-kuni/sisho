package cmd

import (
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/cmd/addCommand"
	"github.com/t-kuni/sisho/cmd/initCommand"
	"github.com/t-kuni/sisho/cmd/makeCommand"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/infrastructure/external/claude"
	"github.com/t-kuni/sisho/infrastructure/external/openAi"
	"github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/infrastructure/repository/file"
	"github.com/t-kuni/sisho/infrastructure/repository/knowledge"
)

type RootCommand struct {
	CobraCommand *cobra.Command
}

func NewRootCommand() *RootCommand {
	cmd := &cobra.Command{
		Use:   "sisho",
		Short: "Sisho is a CLI tool for generating code using LLM",
		Long:  `A CLI tool that uses LLM to generate code based on knowledge sets and project structure.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	fileRepo := file.NewFileRepository()
	configRepo := config.NewConfigRepository()
	knowledgeRepo := knowledge.NewRepository()
	configFindSvc := configFindService.NewConfigFindService(fileRepo)
	contextScanSvc := contextScan.NewContextScanService(fileRepo)
	autoCollectSvc := autoCollect.NewAutoCollectService(configRepo, contextScanSvc)
	knowledgeScanSvc := knowledgeScan.NewKnowledgeScanService(knowledgeRepo, autoCollectSvc)
	knowledgeLoadSvc := knowledgeLoad.NewKnowledgeLoadService(knowledgeRepo)

	claudeClient := claude.NewClaudeClient()
	openAiClient := openAi.NewOpenAIClient()

	initCmd := initCommand.NewInitCommand(configRepo)
	addCmd := addCommand.NewAddCommand(knowledgeRepo)
	makeCmd := makeCommand.NewMakeCommand(
		claudeClient,
		openAiClient,
		configFindSvc,
		configRepo,
		fileRepo,
		autoCollectSvc,
		contextScanSvc,
		knowledgeScanSvc,
		knowledgeLoadSvc,
	)

	cmd.AddCommand(initCmd.CobraCommand)
	cmd.AddCommand(addCmd.CobraCommand)
	cmd.AddCommand(makeCmd.CobraCommand)

	return &RootCommand{
		CobraCommand: cmd,
	}
}
