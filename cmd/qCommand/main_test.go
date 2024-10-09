package qCommand_test

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/cmd/qCommand"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestQCommand(t *testing.T) {
	type Mocks struct {
		Timer          *timer.MockITimer
		ClaudeClient   *claude.MockClient
		OpenAiClient   *openAi.MockClient
		FileRepository *file.MockRepository
		KsuidGenerator *ksuid.MockIKsuid
	}

	callCommand := func(
		mockCtrl *gomock.Controller,
		args []string,
		customizeMocks func(mocks Mocks),
	) error {
		mockTimer := timer.NewMockITimer(mockCtrl)
		mockClaudeClient := claude.NewMockClient(mockCtrl)
		mockOpenAiClient := openAi.NewMockClient(mockCtrl)
		mockFileRepo := file.NewMockRepository(mockCtrl)
		configRepo := config2.NewConfigRepository()
		knowledgeRepo := knowledge2.NewRepository()
		configFindSvc := configFindService.NewConfigFindService(mockFileRepo)
		contextScanSvc := contextScan.NewContextScanService(mockFileRepo)
		autoCollectSvc := autoCollect.NewAutoCollectService(configRepo, contextScanSvc)
		knowledgePathNormalizeSvc := knowledgePathNormalize.NewKnowledgePathNormalizeService()
		knowledgeScanSvc := knowledgeScan.NewKnowledgeScanService(knowledgeRepo, autoCollectSvc, knowledgePathNormalizeSvc)
		knowledgeLoadSvc := knowledgeLoad.NewKnowledgeLoadService(knowledgeRepo)
		mockKsuidGenerator := ksuid.NewMockIKsuid(mockCtrl)
		folderStructureMakeSvc := folderStructureMake.NewFolderStructureMakeService()

		customizeMocks(Mocks{
			ClaudeClient:   mockClaudeClient,
			OpenAiClient:   mockOpenAiClient,
			FileRepository: mockFileRepo,
			Timer:          mockTimer,
			KsuidGenerator: mockKsuidGenerator,
		})

		testee := qCommand.NewQCommand(
			mockClaudeClient,
			mockOpenAiClient,
			configFindSvc,
			configRepo,
			mockFileRepo,
			knowledgeScanSvc,
			knowledgeLoadSvc,
			mockTimer,
			mockKsuidGenerator,
			folderStructureMakeSvc,
		)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(testee.CobraCommand)

		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	t.Run("-iオプションが正常に動作すること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.WriteFile("sisho.yml", []byte(`
llm:
  driver: open-ai
  model: gpt-4
`))

		space.WriteFile("main.go", []byte(`package main`))

		testUtil.Stdin(t, "This is a test question")

		err := callCommand(mockCtrl, []string{"q", "main.go", "-i"}, func(mocks Mocks) {
			mocks.OpenAiClient.EXPECT().SendMessage(gomock.Any(), "gpt-4").
				DoAndReturn(func(messages []openAi.Message, model string) (openAi.GenerationResult, error) {
					assert.Contains(t, messages[0].Content, "main.go")
					assert.Contains(t, messages[0].Content, "package main")
					assert.Contains(t, messages[0].Content, "This is a test question")
					return openAi.GenerationResult{
						Content:           "LLM Response",
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2021-01-02T15:04:05Z")).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").AnyTimes()
		})

		assert.NoError(t, err)
	})

	t.Run("複数のTargetを指定した場合、それらのファイルが読み込まれてプロンプトに記載されること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.WriteFile("sisho.yml", []byte(`
llm:
  driver: open-ai
  model: gpt-4
`))

		space.WriteFile("main.go", []byte(`package main`))
		space.WriteFile("helper.go", []byte(`package helper`))

		err := callCommand(mockCtrl, []string{"q", "main.go", "helper.go"}, func(mocks Mocks) {
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2021-01-02T15:04:05Z")).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").AnyTimes()
			mocks.OpenAiClient.EXPECT().SendMessage(gomock.Any(), "gpt-4").DoAndReturn(func(messages []openAi.Message, model string) (openAi.GenerationResult, error) {
				assert.Contains(t, messages[0].Content, "main.go")
				assert.Contains(t, messages[0].Content, "package main")
				assert.Contains(t, messages[0].Content, "helper.go")
				assert.Contains(t, messages[0].Content, "package helper")
				return openAi.GenerationResult{
					Content:           "LLM Response",
					TerminationReason: "success",
				}, nil
			})
		})

		assert.NoError(t, err)
	})

	t.Run("フォルダ構造情報が正しくプロンプトに含まれること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.WriteFile("sisho.yml", []byte(`
llm:
  driver: open-ai
  model: gpt-4
additional-knowledge:
  folder-structure: true
`))

		space.WriteFile("main.go", []byte(`package main`))
		space.MkDir("subdir")
		space.WriteFile("subdir/helper.go", []byte(`package helper`))

		err := callCommand(mockCtrl, []string{"q", "main.go"}, func(mocks Mocks) {
			mocks.OpenAiClient.EXPECT().SendMessage(gomock.Any(), "gpt-4").
				DoAndReturn(func(messages []openAi.Message, model string) (openAi.GenerationResult, error) {
					assert.Contains(t, messages[0].Content, "main.go")
					assert.Contains(t, messages[0].Content, "package main")
					assert.Contains(t, messages[0].Content, "/subdir")
					assert.Contains(t, messages[0].Content, "helper.go")
					return openAi.GenerationResult{
						Content:           "LLM Response",
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2021-01-02T15:04:05Z")).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").AnyTimes()
		})

		assert.NoError(t, err)
	})

	t.Run("knowledgeスキャンとロードが正しく行われること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.WriteFile("sisho.yml", []byte(`
llm:
  driver: open-ai
  model: gpt-4
`))

		space.WriteFile("main.go", []byte(`package main`))
		space.WriteFile(".knowledge.yml", []byte(`
knowledge:
  - path: example.go
    kind: examples
`))
		space.WriteFile("example.go", []byte(`package example`))

		err := callCommand(mockCtrl, []string{"q", "main.go"}, func(mocks Mocks) {
			mocks.OpenAiClient.EXPECT().SendMessage(gomock.Any(), "gpt-4").
				DoAndReturn(func(messages []openAi.Message, model string) (openAi.GenerationResult, error) {
					assert.Contains(t, messages[0].Content, "main.go")
					assert.Contains(t, messages[0].Content, "package main")
					assert.Contains(t, messages[0].Content, "example.go")
					assert.Contains(t, messages[0].Content, "package example")
					return openAi.GenerationResult{
						Content:           "LLM Response",
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2021-01-02T15:04:05Z")).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").AnyTimes()
		})

		assert.NoError(t, err)
	})
}
