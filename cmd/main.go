package cmd

import (
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/cmd/addCommand"
	"github.com/t-kuni/sisho/cmd/depsGraphCommand"
	"github.com/t-kuni/sisho/cmd/extractCommand"
	"github.com/t-kuni/sisho/cmd/initCommand"
	"github.com/t-kuni/sisho/cmd/makeCommand"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/service/projectScan"
	"github.com/t-kuni/sisho/infrastructure/external/claude"
	"github.com/t-kuni/sisho/infrastructure/external/openAi"
	"github.com/t-kuni/sisho/infrastructure/repository/config"
	depsGraph2 "github.com/t-kuni/sisho/infrastructure/repository/depsGraph"
	"github.com/t-kuni/sisho/infrastructure/repository/file"
	"github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/infrastructure/system/ksuid"
	"github.com/t-kuni/sisho/infrastructure/system/timer"
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
	depsGraphRepo := depsGraph2.NewRepository()
	ksuidGenerator := ksuid.NewKsuidGenerator()
	configFindSvc := configFindService.NewConfigFindService(fileRepo)
	contextScanSvc := contextScan.NewContextScanService(fileRepo)
	autoCollectSvc := autoCollect.NewAutoCollectService(configRepo, contextScanSvc)
	knowledgePathNormalizeSvc := knowledgePathNormalize.NewKnowledgePathNormalizeService()
	knowledgeScanSvc := knowledgeScan.NewKnowledgeScanService(knowledgeRepo, autoCollectSvc, knowledgePathNormalizeSvc)
	knowledgeLoadSvc := knowledgeLoad.NewKnowledgeLoadService(knowledgeRepo)
	projectScanSvc := projectScan.NewProjectScanService(fileRepo)
	folderStructureMakeSvc := folderStructureMake.NewFolderStructureMakeService()

	claudeClient := claude.NewClaudeClient()
	openAiClient := openAi.NewOpenAIClient()

	initCmd := initCommand.NewInitCommand(configRepo, fileRepo)
	addCmd := addCommand.NewAddCommand(knowledgeRepo)
	makeCmd := makeCommand.NewMakeCommand(
		claudeClient,
		openAiClient,
		configFindSvc,
		configRepo,
		fileRepo,
		knowledgeScanSvc,
		knowledgeLoadSvc,
		depsGraphRepo,
		timer.NewTimer(),
		ksuidGenerator,
		folderStructureMakeSvc,
	)
	extractCmd := extractCommand.NewExtractCommand(
		claudeClient,
		openAiClient,
		configFindSvc,
		configRepo,
		fileRepo,
		knowledgeRepo,
		folderStructureMakeSvc,
		knowledgePathNormalizeSvc,
	)
	depsGraphCmd := depsGraphCommand.NewDepsGraphCommand(
		configFindSvc,
		projectScanSvc,
		knowledgeRepo,
		depsGraphRepo,
		knowledgePathNormalizeSvc,
	)

	cmd.AddCommand(initCmd.CobraCommand)
	cmd.AddCommand(addCmd.CobraCommand)
	cmd.AddCommand(makeCmd.CobraCommand)
	cmd.AddCommand(extractCmd.CobraCommand)
	cmd.AddCommand(depsGraphCmd.CobraCommand)

	return &RootCommand{
		CobraCommand: cmd,
	}
}
