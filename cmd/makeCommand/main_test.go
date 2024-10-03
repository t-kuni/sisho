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
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/infrastructure/repository/depsGraph"
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
		knowledgeScanSvc := knowledgeScan.NewKnowledgeScanService(knowledgeRepo, autoCollectSvc)
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

		makeCmd := NewMakeCommand(
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
		)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(makeCmd.CobraCommand)

		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	t.Run("指定したファイルに生成されたコードが反映されること", func(t *testing.T) {
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
					assert.Contains(t, messages[0].Content, "aaa/bbb/ccc/ddd.txt")
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

	t.Run("Knowledgeスキャンが正しく行われること", func(t *testing.T) {
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

		space.WriteFile("aaa/bbb/SPEC.md", []byte("This is SPEC.md"))
		space.WriteFile("aaa/bbb/ccc/.knowledge.yml", []byte(`
knowledge:
  - path: ../SPEC.md
    kind: specifications
`))

		space.WriteFile("aaa/bbb/SPEC2.md", []byte("This is SPEC2.md"))
		space.WriteFile("aaa/.knowledge.yml", []byte(`
knowledge:
  - path: bbb/SPEC2.md
    kind: specifications
`))

		generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
					content := messages[0].Content
					// Check if knowledge from .knowledge.yml is included
					assert.Contains(t, content, "aaa/bbb/SPEC.md")
					assert.Contains(t, content, "This is SPEC.md")
					assert.Contains(t, content, "aaa/bbb/SPEC2.md")
					assert.Contains(t, content, "This is SPEC2.md")
					return generated, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)
	})

	t.Run("単一ファイル知識リストファイルが正しく読み込まれること", func(t *testing.T) {
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

		space.WriteFile("aaa/bbb/ccc/SPEC.md", []byte("This is SPEC.md"))
		space.WriteFile("aaa/bbb/ccc/ddd.txt.know.yml", []byte(`
knowledge:
  - path: SPEC.md
    kind: specifications
`))

		generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
					content := messages[0].Content
					assert.Contains(t, content, "aaa/bbb/ccc/SPEC.md")
					assert.Contains(t, content, "This is SPEC.md")
					return generated, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)
	})

	t.Run("[TARGET_CODE].mdが正しく読み込まれること", func(t *testing.T) {
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
		space.WriteFile("aaa/bbb/ccc/ddd.txt.md", []byte("This is ddd.txt.md"))

		generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
					content := messages[0].Content
					assert.Contains(t, content, "aaa/bbb/ccc/ddd.txt.md")
					assert.Contains(t, content, "This is ddd.txt.md")
					return generated, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)
	})

	t.Run("Target Codeと同階層ではない[TARGET_CODE].mdは無視されること", func(t *testing.T) {
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
		space.WriteFile("aaa/bbb/ddd.txt.md", []byte("This is ddd.txt.md"))

		generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		err := callCommand(mockCtrl, []string{"make", "aaa/bbb/ccc/ddd.txt", "-a"}, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (string, error) {
					content := messages[0].Content
					assert.NotContains(t, content, "aaa/bbb/ddd.txt.md")
					assert.NotContains(t, content, "This is ddd.txt.md")
					return generated, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		assert.NoError(t, err)
	})

	t.Run("連鎖的生成(-cオプション)について", func(t *testing.T) {
		t.Run("連鎖的生成(-cオプション)が正常に動作すること", func(t *testing.T) {
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
					DoAndReturn(func(messages []claude.Message, model string) (string, error) {
						//assert.Contains(t, messages[0].Content, "FILE3_CONTENT")
						return fmt.Sprintf(generatedFormat, "file3.go", 1), nil
					})
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (string, error) {
						//assert.Contains(t, messages[0].Content, "FILE2_CONTENT")
						return fmt.Sprintf(generatedFormat, "file2.go", 2), nil
					})
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (string, error) {
						//assert.Contains(t, messages[0].Content, "FILE1_CONTENT")
						return fmt.Sprintf(generatedFormat, "file1.go", 3), nil
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

		t.Run("deps-graphが存在しない場合はエラーを返すこと", func(t *testing.T) {
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
			space.WriteFile("file1.go", []byte("FILE1_CONTENT"))

			err := callCommand(mockCtrl, []string{"make", "file1.go", "-c"}, func(mocks Mocks) {
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			})

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to read deps-graph.json")
		})

		t.Run("deps-graphに記載のないファイルは一番深い深度のTarget Codeとして扱うこと", func(t *testing.T) {
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
			space.WriteFile("file1.go", []byte("FILE1_CONTENT"))
			space.WriteFile("file2.go", []byte("FILE2_CONTENT"))
			space.WriteFile("file3.go", []byte("FILE3_CONTENT"))
			space.WriteFile(".sisho/deps-graph.json", []byte(`
{
  "file2.go": [ "file1.go" ]
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
					DoAndReturn(func(messages []claude.Message, model string) (string, error) {
						return fmt.Sprintf(generatedFormat, "file3.go", 1), nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			assert.NoError(t, err)

			// Assert
			space.AssertFile("file3.go", func(actual []byte) {
				assert.Equal(t, "UPDATED_CONTENT1", string(actual))
			})
		})
	})

	t.Run("フォルダ構成情報のプロンプト出力について", func(t *testing.T) {
		t.Run("正しく出力されること", func(t *testing.T) {
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
			space.WriteFile("file1.go", []byte("FILE1_CONTENT"))
			space.WriteFile("dir1/file2.go", []byte("FILE2_CONTENT"))
			space.WriteFile("dir1/dir2/file3.go", []byte("FILE3_CONTENT"))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `file1.go
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`

			err := callCommand(mockCtrl, []string{"make", "file1.go", "-a"}, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (string, error) {
						content := messages[0].Content
						assert.Contains(t, content, "# Folder Structure")
						assert.Contains(t, content, "file1.go")
						assert.Contains(t, content, "/dir1")
						assert.Contains(t, content, "file2.go")
						assert.Contains(t, content, "/dir2")
						assert.Contains(t, content, "file3.go")
						return generated, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			assert.NoError(t, err)
		})

		t.Run(".sishoignoreに記載のあるファイルは無視されること", func(t *testing.T) {
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
			space.WriteFile("file1.go", []byte("FILE1_CONTENT"))
			space.WriteFile("dir1/file2.go", []byte("FILE2_CONTENT"))
			space.WriteFile("dir1/dir2/file3.go", []byte("FILE3_CONTENT"))
			space.WriteFile("ignore_this.txt", []byte("IGNORE_CONTENT"))
			space.WriteFile(".sishoignore", []byte("ignore_this.txt\ndir1/dir2"))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `file1.go
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`

			err := callCommand(mockCtrl, []string{"make", "file1.go", "-a"}, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (string, error) {
						content := messages[0].Content
						assert.Contains(t, content, "# Folder Structure")
						assert.Contains(t, content, "file1.go")
						assert.Contains(t, content, "/dir1")
						assert.Contains(t, content, "file2.go")
						assert.NotContains(t, content, "ignore_this.txt")
						assert.NotContains(t, content, "/dir2")
						assert.NotContains(t, content, "file3.go")
						return generated, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			assert.NoError(t, err)
		})
	})
}
