package extractCommand

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestExtractCommand(t *testing.T) {
	type Mocks struct {
		ClaudeClient   *claude.MockClient
		OpenAiClient   *openAi.MockClient
		FileRepository *file.MockRepository
	}

	callCommand := func(
		mockCtrl *gomock.Controller,
		args []string,
		customizeMocks func(mocks Mocks),
	) error {
		mockClaudeClient := claude.NewMockClient(mockCtrl)
		mockOpenAiClient := openAi.NewMockClient(mockCtrl)
		mockFileRepo := file.NewMockRepository(mockCtrl)
		configRepo := config2.NewConfigRepository()
		knowledgeRepo := knowledge2.NewRepository()
		configFindSvc := configFindService.NewConfigFindService(mockFileRepo)

		customizeMocks(Mocks{
			ClaudeClient:   mockClaudeClient,
			OpenAiClient:   mockOpenAiClient,
			FileRepository: mockFileRepo,
		})

		extractCmd := NewExtractCommand(
			mockClaudeClient,
			mockOpenAiClient,
			configFindSvc,
			configRepo,
			mockFileRepo,
			knowledgeRepo,
		)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(extractCmd.CobraCommand)

		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	t.Run("正常系: 知識リストが正しく抽出されること", func(t *testing.T) {
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
`))
		space.WriteFile("target.go", []byte("package main\n\nfunc main() {}"))

		generatedKnowledge := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `target.go.know.yml
knowledge:
  - path: some/path/file1.go
    kind: examples
  - path: another/path/file2.go
    kind: implementations
    chain-make: true
  - path: another/path/file3.go
    kind: implementations
` + "```" + `<!-- CODE_BLOCK_END -->
`

		err := callCommand(mockCtrl, []string{"extract", "target.go"}, func(mocks Mocks) {
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(generatedKnowledge, nil)
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
		})

		assert.NoError(t, err)

		// Assert
		space.AssertFile("target.go.know.yml", func(actual []byte) {
			expectedContent := `knowledge:
  - path: some/path/file1.go
    kind: examples
  - path: another/path/file2.go
    kind: implementations
    chain-make: true
  - path: another/path/file3.go
    kind: implementations
`
			assert.YAMLEq(t, expectedContent, string(actual))
		})
	})

	t.Run("正常系: 既存の知識リストとマージされること", func(t *testing.T) {
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
`))
		space.WriteFile("target.go", []byte("package main\n\nfunc main() {}"))
		space.WriteFile("target.go.know.yml", []byte(`
knowledge:
  - path: existing/file.go
    kind: specifications
`))

		generatedKnowledge := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `target.go.know.yml
knowledge:
  - path: some/path/file1.go
    kind: examples
  - path: another/path/file2.go
    kind: implementations
` + "```" + `<!-- CODE_BLOCK_END -->
`

		err := callCommand(mockCtrl, []string{"extract", "target.go"}, func(mocks Mocks) {
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(generatedKnowledge, nil)
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
		})

		assert.NoError(t, err)

		// Assert
		space.AssertFile("target.go.know.yml", func(actual []byte) {
			expectedContent := `knowledge:
  - path: existing/file.go
    kind: specifications
  - path: some/path/file1.go
    kind: examples
  - path: another/path/file2.go
    kind: implementations
`
			assert.YAMLEq(t, expectedContent, string(actual))
		})
	})

	t.Run("正常系: フォルダ構造情報が正しく含まれること", func(t *testing.T) {
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
additional-knowledge:
    folder-structure: true
`))
		space.WriteFile("target.go", []byte("package main\n\nfunc main() {}"))
		space.WriteFile("dir1/file1.go", []byte(""))
		space.WriteFile("dir2/subdir/file2.go", []byte(""))

		generatedKnowledge := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `target.go.know.yml
knowledge:
  - path: some/path/file1.go
    kind: examples
` + "```" + `<!-- CODE_BLOCK_END -->
`

		var capturedPrompt string

		err := callCommand(mockCtrl, []string{"extract", "target.go"}, func(mocks Mocks) {
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
					capturedPrompt = messages[0].Content
					return generatedKnowledge, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
		})

		assert.NoError(t, err)

		// Assert folder structure in the prompt
		assert.Contains(t, capturedPrompt, "# Folder Structure")
		assert.Contains(t, capturedPrompt, "target.go")
		assert.Contains(t, capturedPrompt, "/dir1")
		assert.Contains(t, capturedPrompt, "file1.go")
		assert.Contains(t, capturedPrompt, "/dir2")
		assert.Contains(t, capturedPrompt, "/subdir")
		assert.Contains(t, capturedPrompt, "file2.go")
	})

	//	t.Run("正常系: .sishoignoreに記載されたファイルがフォルダ構造情報から除外されること", func(t *testing.T) {
	//		mockCtrl := gomock.NewController(t)
	//		defer mockCtrl.Finish()
	//
	//		space := testUtil.BeginTestSpace(t)
	//		defer space.CleanUp()
	//
	//		// Setup Files
	//		space.WriteFile("sisho.yml", []byte(`
	//lang: ja
	//llm:
	//    driver: anthropic
	//    model: claude-3-5-sonnet-20240620
	//additional-knowledge:
	//    folder-structure: true
	//`))
	//		space.WriteFile("target.go", []byte("package main\n\nfunc main() {}"))
	//		space.WriteFile("dir1/file1.go", []byte(""))
	//		space.WriteFile("dir2/subdir/file2.go", []byte(""))
	//		space.WriteFile("ignore_this.txt", []byte(""))
	//		space.WriteFile(".sishoignore", []byte("ignore_this.txt\ndir2"))
	//
	//		generatedKnowledge := `
	//<!-- CODE_BLOCK_BEGIN -->` + "```" + `target.go.know.yml
	//knowledge:
	//  - path: some/path/file1.go
	//    kind: examples
	//` + "```" + `<!-- CODE_BLOCK_END -->
	//`
	//
	//		var capturedPrompt string
	//
	//		err := callCommand(mockCtrl, []string{"extract", "target.go"}, func(mocks Mocks) {
	//			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
	//				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
	//					capturedPrompt = messages[0].Content
	//					return generatedKnowledge, nil
	//				})
	//			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
	//		})
	//
	//		assert.NoError(t, err)
	//
	//		// Assert folder structure in the prompt
	//		assert.Contains(t, capturedPrompt, "# Folder Structure")
	//		assert.Contains(t, capturedPrompt, "target.go")
	//		assert.Contains(t, capturedPrompt, "/dir1")
	//		assert.Contains(t, capturedPrompt, "file1.go")
	//		assert.NotContains(t, capturedPrompt, "ignore_this.txt")
	//		assert.NotContains(t, capturedPrompt, "/dir2")
	//		assert.NotContains(t, capturedPrompt, "/subdir")
	//		assert.NotContains(t, capturedPrompt, "file2.go")
	//	})
}
