package cmd

import (
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/cmd/addCommand"
	"github.com/t-kuni/sisho/cmd/depsGraphCommand"
	"github.com/t-kuni/sisho/cmd/extractCommand"
	"github.com/t-kuni/sisho/cmd/fixTaskCommand"
	"github.com/t-kuni/sisho/cmd/initCommand"
	"github.com/t-kuni/sisho/cmd/makeCommand"
	"github.com/t-kuni/sisho/cmd/qCommand"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/chatFactory"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/extractCodeBlock"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/service/make"
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
	extractCodeBlockSvc := extractCodeBlock.NewCodeBlockExtractService()

	claudeClient := claude.NewClaudeClient()
	openAiClient := openAi.NewOpenAIClient()
	chatFactory := chatFactory.NewChatFactory(openAiClient, claudeClient)

	initCmd := initCommand.NewInitCommand(configRepo, fileRepo)
	addCmd := addCommand.NewAddCommand(knowledgeRepo)
	makeService := make.NewMakeService(
		configFindSvc,
		configRepo,
		knowledgeScanSvc,
		knowledgeLoadSvc,
		depsGraphRepo,
		timer.NewTimer(),
		ksuidGenerator,
		folderStructureMakeSvc,
		extractCodeBlockSvc,
		chatFactory,
	)
	makeCmd := makeCommand.NewMakeCommand(makeService)
	extractCmd := extractCommand.NewExtractCommand(
		configFindSvc,
		configRepo,
		knowledgeRepo,
		folderStructureMakeSvc,
		knowledgePathNormalizeSvc,
		extractCodeBlockSvc,
		chatFactory,
	)
	depsGraphCmd := depsGraphCommand.NewDepsGraphCommand(
		configFindSvc,
		projectScanSvc,
		knowledgeRepo,
		depsGraphRepo,
		knowledgePathNormalizeSvc,
	)
	qCmd := qCommand.NewQCommand(
		configFindSvc,
		configRepo,
		knowledgeScanSvc,
		knowledgeLoadSvc,
		timer.NewTimer(),
		ksuidGenerator,
		folderStructureMakeSvc,
		chatFactory,
	)
	fixTaskCmd := fixTaskCommand.NewFixTaskCommand(
		configFindSvc,
		configRepo,
		makeService,
		chatFactory,
		timer.NewTimer(),
		ksuidGenerator,
		folderStructureMakeSvc,
		extractCodeBlockSvc,
	)

	cmd.AddCommand(initCmd.CobraCommand)
	cmd.AddCommand(addCmd.CobraCommand)
	cmd.AddCommand(makeCmd.CobraCommand)
	cmd.AddCommand(extractCmd.CobraCommand)
	cmd.AddCommand(depsGraphCmd.CobraCommand)
	cmd.AddCommand(qCmd.CobraCommand)
	cmd.AddCommand(fixTaskCmd.CobraCommand)

	return &RootCommand{
		CobraCommand: cmd,
	}
}
