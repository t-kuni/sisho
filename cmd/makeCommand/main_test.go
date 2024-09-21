package makeCommand

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"path/filepath"
	"testing"
)

func TestMakeCommand(t *testing.T) {
	type Mocks struct {
		Timer          *timer.MockITimer
		ClaudeClient   *claude.MockClient
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
		knowledgeScanSvc := knowledgeScan.NewKnowledgeScanService(knowledgeRepo, autoCollectSvc)
		knowledgeLoadSvc := knowledgeLoad.NewKnowledgeLoadService(knowledgeRepo)
		mockKsuidGenerator := ksuid.NewMockIKsuid(mockCtrl)

		customizeMocks(Mocks{
			ClaudeClient:   mockClaudeClient,
			FileRepository: mockFileRepo,
			Timer:          mockTimer,
			KsuidGenerator: mockKsuidGenerator,
		})

		makeCmd := NewMakeCommand(
			mockClaudeClient,
			mockOpenAiClient,
			configFindSvc,
			configRepo,
			mockFileRepo,
			knowledgeScanSvc,
			knowledgeLoadSvc,
			mockTimer,
			mockKsuidGenerator,
		)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(makeCmd.CobraCommand)

		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	t.Run("指定したファイルに生成されたコードが反映されること", func(t *testing.T) {
		// 指定したコードがプロンプトに埋め込まれること

		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		config := `
lang: ja
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
auto-collect:
    README.md: true
    "[TARGET_CODE].md": true
additional-knowledge:
    folder-structure: true
`
		space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))
		space.WriteFile("sisho.yml", []byte(config))

		generated := `
dummy text

<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->

dummy text
`

		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
					assert.Contains(t, messages[0].Content, "CURRENT_CONTENT")
					return generated, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)

		// Assert
		space.AssertFile("aaa/bbb/ccc/ddd.txt", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT", string(actual))
		})
	})

	t.Run("履歴が保存されること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		config := `
lang: ja
llm:
   driver: anthropic
   model: claude-3-5-sonnet-20240620
auto-collect:
   README.md: true
   "[TARGET_CODE].md": true
additional-knowledge:
   folder-structure: true
`
		space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))
		space.WriteFile("sisho.yml", []byte(config))

		generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(generated, nil)
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)

		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "2022-01-01T00:00:00"))
		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "prompt.md"))
		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "answer.md"))
	})
}
