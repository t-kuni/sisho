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
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"path/filepath"
	"testing"
)

func TestMakeCommand(t *testing.T) {
	t.Run("makeコマンドが正常に実行されること", func(t *testing.T) {
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

		// Setup Mocks
		mockTimer := timer.NewMockITimer(mockCtrl)
		mockTimer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
		mockClaudeClient := claude.NewMockClient(mockCtrl)

		generated := `
dummy text

<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->

dummy text
`
		mockClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
			DoAndReturn(func(messages []claude.Message, model string) (string, error) {
				assert.Contains(t, messages[0].Content, "CURRENT_CONTENT")
				return generated, nil
			})

		mockOpenAiClient := openAi.NewMockClient(mockCtrl)
		mockFileRepo := file.NewMockRepository(mockCtrl)
		mockFileRepo.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
		configRepo := config2.NewConfigRepository()
		knowledgeRepo := knowledge.NewRepository()
		configFindSvc := configFindService.NewConfigFindService(mockFileRepo)
		contextScanSvc := contextScan.NewContextScanService(mockFileRepo)
		autoCollectSvc := autoCollect.NewAutoCollectService(configRepo, contextScanSvc)
		knowledgeScanSvc := knowledgeScan.NewKnowledgeScanService(knowledgeRepo, autoCollectSvc)
		knowledgeLoadSvc := knowledgeLoad.NewKnowledgeLoadService(knowledgeRepo)

		makeCmd := NewMakeCommand(
			mockClaudeClient,
			mockOpenAiClient,
			configFindSvc,
			configRepo,
			mockFileRepo,
			autoCollectSvc,
			contextScanSvc,
			knowledgeScanSvc,
			knowledgeLoadSvc,
			mockTimer,
		)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(makeCmd.CobraCommand)

		rootCmd.SetArgs([]string{"make", "aaa/bbb/ccc/ddd.txt", "-a"})
		err := rootCmd.Execute()
		assert.NoError(t, err)

		// Assert
		space.AssertFile("aaa/bbb/ccc/ddd.txt", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT", string(actual))
		})

		space.AssertExistPath(filepath.Join(".sisho", "history"))

		//historySubDir := filepath.Join(historyDir, entries[0].Name())
		//space.AssertExistPath(filepath.Join(historySubDir, "prompt.md"))
		//space.AssertExistPath(filepath.Join(historySubDir, "answer.md"))
	})
}
