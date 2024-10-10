package makeCommand

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"github.com/t-kuni/sisho/domain/service/extractCodeBlock"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	makeService "github.com/t-kuni/sisho/domain/service/make"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/infrastructure/repository/depsGraph"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestMakeCommand(t *testing.T) {
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

		customizeMocks(Mocks{
			ClaudeClient:   mockClaudeClient,
			OpenAiClient:   mockOpenAiClient,
			FileRepository: mockFileRepo,
			Timer:          mockTimer,
			KsuidGenerator: mockKsuidGenerator,
		})

		makeSvc := makeService.NewMakeService(
			mockClaudeClient,
			mockOpenAiClient,
			configFindSvc,
			configRepo,
			mockFileRepo,
			knowledgeScanSvc,
			knowledgeLoadSvc,
			depsGraphRepo,
			mockTimer,
			mockKsuidGenerator,
			folderStructureMakeSvc,
			extractCodeBlockSvc,
		)
		makeCmd := NewMakeCommand(makeSvc)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(makeCmd.CobraCommand)

		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	t.Run("複数のファイルを指定して生成されたコードが反映されること(aオプションの検証)", func(t *testing.T) {
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
		space.WriteFile("aaa/bbb.txt", []byte("CURRENT_CONTENT1"))
		space.WriteFile("aaa/ccc.txt", []byte("CURRENT_CONTENT2"))

		generatedTmpl := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `%s
UPDATED_CONTENT%d
` + "```" + `<!-- CODE_BLOCK_END -->
`

		err := callCommand(mockCtrl, []string{"make", "aaa/bbb.txt", "aaa/ccc.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					assert.Len(t, messages, 1)
					assert.Contains(t, messages[0].Content, "aaa/bbb.txt")
					assert.Contains(t, messages[0].Content, "CURRENT_CONTENT1")
					assert.Contains(t, messages[0].Content, "aaa/ccc.txt")
					assert.Contains(t, messages[0].Content, "CURRENT_CONTENT2")
					return claude.GenerationResult{
						Content:           fmt.Sprintf(generatedTmpl, "aaa/bbb.txt", 1),
						TerminationReason: "success",
					}, nil
				})
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					assert.Len(t, messages, 1)
					assert.Contains(t, messages[0].Content, "aaa/bbb.txt")
					assert.Contains(t, messages[0].Content, "UPDATED_CONTENT1")
					assert.Contains(t, messages[0].Content, "aaa/ccc.txt")
					assert.Contains(t, messages[0].Content, "CURRENT_CONTENT2")
					return claude.GenerationResult{
						Content:           fmt.Sprintf(generatedTmpl, "aaa/ccc.txt", 2),
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)

		// Assert
		space.AssertFile("aaa/bbb.txt", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT1", string(actual))
		})
		space.AssertFile("aaa/ccc.txt", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT2", string(actual))
		})
	})

	t.Run("標準入力から入力されたテキストがプロンプトのAdditional Instructionに反映されること(iオプションの検証)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
lang: ja
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
auto-collect:
    README.md: true
    "[TARGET_CODE].md": true
additional-knowledge:
    folder-structure: true
`))
		space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

		inputText := "標準入力からのテキスト\n次の行も含む"
		testUtil.Stdin(t, inputText)

		generated := `
dummy text

<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->

dummy text
`

		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-ai"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					assert.Contains(t, messages[0].Content, "Additional Instruction")
					assert.Contains(t, messages[0].Content, inputText)
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)
	})

	t.Run("連鎖的生成が正常に動作すること(cオプションの検証)", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
lang: ja
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
auto-collect:
    README.md: true
    "[TARGET_CODE].md": true
additional-knowledge:
    folder-structure: true
`))
		space.WriteFile("fiel1.go", []byte("FILE1_CONTENT"))
		space.WriteFile("fiel2.go", []byte("FILE2_CONTENT"))
		space.WriteFile("fiel3.go", []byte("FILE3_CONTENT"))
		space.WriteFile(".sisho/deps-graph.json", []byte(`
{
  "file2.go": [ "file1.go" ],
  "file3.go": [ "file2.go" ]
}
`))

		generatedFormat := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `%s
UPDATED_CONTENT%d
` + "```" + `<!-- CODE_BLOCK_END -->
`

		err := callCommand(mockCtrl, []string{"make", "file3.go", "-ac"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					//assert.Contains(t, messages[0].Content, "FILE3_CONTENT")
					return claude.GenerationResult{
						Content:           fmt.Sprintf(generatedFormat, "file3.go", 1),
						TerminationReason: "success",
					}, nil
				})
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					//assert.Contains(t, messages[0].Content, "FILE2_CONTENT")
					return claude.GenerationResult{
						Content:           fmt.Sprintf(generatedFormat, "file2.go", 2),
						TerminationReason: "success",
					}, nil
				})
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					//assert.Contains(t, messages[0].Content, "FILE1_CONTENT")
					return claude.GenerationResult{
						Content:           fmt.Sprintf(generatedFormat, "file1.go", 3),
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)

		// Assert
		space.AssertFile("file1.go", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT3", string(actual))
		})
		space.AssertFile("file2.go", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT2", string(actual))
		})
		space.AssertFile("file3.go", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT1", string(actual))
		})
	})
}
