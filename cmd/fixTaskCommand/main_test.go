package fixTaskCommand

import (
	"bytes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/model/chat"
	"github.com/t-kuni/sisho/domain/repository/file"
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
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/infrastructure/repository/depsGraph"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestFixTaskCommand(t *testing.T) {
	type Mocks struct {
		Timer          *timer.MockITimer
		ClaudeClient   *claude.MockClient
		OpenAiClient   *openAi.MockClient
		FileRepository *file.MockRepository
		KsuidGenerator *ksuid.MockIKsuid
		Chat           *chat.MockChat
	}

	callCommand := func(
		mockCtrl *gomock.Controller,
		args []string,
		customizeMocks func(mocks Mocks),
	) (string, error) {
		mockTimer := timer.NewMockITimer(mockCtrl)
		mockClaudeClient := claude.NewMockClient(mockCtrl)
		mockOpenAiClient := openAi.NewMockClient(mockCtrl)
		mockFileRepo := file.NewMockRepository(mockCtrl)
		depsGraphRepo := depsGraph.NewRepository()
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
		extractCodeBlockSvc := extractCodeBlock.NewCodeBlockExtractService()
		mockChat := chat.NewMockChat(mockCtrl)
		chatFactorySvc := chatFactory.NewChatFactory(mockOpenAiClient, mockClaudeClient)

		customizeMocks(Mocks{
			ClaudeClient:   mockClaudeClient,
			OpenAiClient:   mockOpenAiClient,
			FileRepository: mockFileRepo,
			Timer:          mockTimer,
			KsuidGenerator: mockKsuidGenerator,
			Chat:           mockChat,
		})

		makeSvc := make.NewMakeService(
			configFindSvc,
			configRepo,
			knowledgeScanSvc,
			knowledgeLoadSvc,
			depsGraphRepo,
			mockTimer,
			mockKsuidGenerator,
			folderStructureMakeSvc,
			extractCodeBlockSvc,
			chatFactorySvc,
		)
		fixTaskCmd := NewFixTaskCommand(
			configFindSvc,
			configRepo,
			makeSvc,
			chatFactorySvc,
			mockTimer,
			mockKsuidGenerator,
			folderStructureMakeSvc,
			extractCodeBlockSvc,
		)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(fixTaskCmd.CobraCommand)

		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs(args)

		err := rootCmd.Execute()
		return buf.String(), err
	}

	t.Run("正常に動作すること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
tasks:
  - name: test-task
    run: |
      (>&2 echo "エラーメッセージ") && exit 1
`))
		space.WriteFile("aaa/bbb.txt", []byte("CURRENT_CONTENT"))

		_, err := callCommand(mockCtrl, []string{"fix:task", "test-task"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					assert.Contains(t, messages[0].Content, "Stderr:\nエラーメッセージ")
					assert.Contains(t, messages[0].Content, "(>&2 echo \"エラーメッセージ\") && exit 1")
					generated := "<!-- CODE_BLOCK_BEGIN -->```json" + `
[ "aaa/bbb.txt" ]
` + "```<!-- CODE_BLOCK_END -->"
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					assert.Contains(t, messages[0].Content, "エラーメッセージ")
					generated := "<!-- CODE_BLOCK_BEGIN -->```aaa/bbb.txt" + `
UPDATED_CONTENT
` + "```<!-- CODE_BLOCK_END -->"
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").Times(2)
		})
		assert.Contains(t, err.Error(), "エラーメッセージ")

		// Assert
		space.AssertFile("aaa/bbb.txt", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT", string(actual))
		})
	})

	t.Run("フォルダ構造情報がプロンプトに含まれること", func(t *testing.T) {
		// パスが検出出来なかった場合makeに進まず終了すること

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
additional-knowledge:
    folder-structure: true
tasks:
  - name: test-task
    run: |
      (>&2 echo "エラーメッセージ") && exit 1
`))
		space.WriteFile("aaa/bbb.txt", []byte("CURRENT_CONTENT"))
		space.WriteFile("ccc/ddd.txt", []byte("CURRENT_CONTENT"))

		_, err := callCommand(mockCtrl, []string{"fix:task", "test-task"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					generated := "<!-- CODE_BLOCK_BEGIN -->```json\n[]\n```<!-- CODE_BLOCK_END -->"
					assert.Contains(t, messages[0].Content, "aaa")
					assert.Contains(t, messages[0].Content, "bbb.txt")
					assert.Contains(t, messages[0].Content, "ccc")
					assert.Contains(t, messages[0].Content, "ddd.txt")
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").Times(1)
		})
		assert.NotNil(t, err, "Need to fail")

		// Assert
		space.AssertFile("aaa/bbb.txt", func(actual []byte) {
			assert.Equal(t, "CURRENT_CONTENT", string(actual))
		})
	})

	t.Run("taskが成功した場合終了すること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
tasks:
  - name: test-task
    run: |
      echo "成功"
`))
		space.WriteFile("aaa/bbb.txt", []byte("CURRENT_CONTENT"))

		_, err := callCommand(mockCtrl, []string{"fix:task", "test-task"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid").Times(1)
		})
		assert.NoError(t, err)

		// Assert
		space.AssertFile("aaa/bbb.txt", func(actual []byte) {
			assert.Equal(t, "CURRENT_CONTENT", string(actual))
		})
	})

	t.Run("存在しないtaskの場合何もせずに終了すること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
`))
		space.WriteFile("aaa/bbb.txt", []byte("CURRENT_CONTENT"))

		_, err := callCommand(mockCtrl, []string{"fix:task", "test-task"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
		})
		assert.NotNil(t, err, "Need to fail")

		// Assert
		space.AssertFile("aaa/bbb.txt", func(actual []byte) {
			assert.Equal(t, "CURRENT_CONTENT", string(actual))
		})
	})
}
