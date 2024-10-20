package make_test

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
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
	makeService "github.com/t-kuni/sisho/domain/service/make"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	config2 "github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/infrastructure/repository/depsGraph"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"path/filepath"
	"strings"
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

	factory := func(
		mockCtrl *gomock.Controller,
		customizeMocks func(mocks Mocks),
	) *makeService.MakeService {
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
		chatFactory := chatFactory.NewChatFactory(mockOpenAiClient, mockClaudeClient)

		customizeMocks(Mocks{
			ClaudeClient:   mockClaudeClient,
			OpenAiClient:   mockOpenAiClient,
			FileRepository: mockFileRepo,
			Timer:          mockTimer,
			KsuidGenerator: mockKsuidGenerator,
		})

		return makeService.NewMakeService(
			configFindSvc,
			configRepo,
			knowledgeScanSvc,
			knowledgeLoadSvc,
			depsGraphRepo,
			mockTimer,
			mockKsuidGenerator,
			folderStructureMakeSvc,
			extractCodeBlockSvc,
			chatFactory,
		)
	}

	t.Run("指定したファイルに生成されたコードが反映されること", func(t *testing.T) {
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
		space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

		generated := `
dummy text

<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->

dummy text
`

		testee := factory(mockCtrl, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					assert.NotContains(t, messages[0].Content, space.Dir)
					assert.Contains(t, messages[0].Content, "aaa/bbb/ccc/ddd.txt")
					assert.Contains(t, messages[0].Content, "CURRENT_CONTENT")
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
		assert.NoError(t, err)

		// Assert
		space.AssertFile("aaa/bbb/ccc/ddd.txt", func(actual []byte) {
			assert.Equal(t, "UPDATED_CONTENT", string(actual))
		})
	})

	t.Run("dryRunフラグがtrueの場合LLMによるファイル生成が行われないこと", func(t *testing.T) {
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
		space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

		testee := factory(mockCtrl, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", true)
		assert.NoError(t, err)

		// Assert
		space.AssertFile("aaa/bbb/ccc/ddd.txt", func(actual []byte) {
			assert.Equal(t, "CURRENT_CONTENT", string(actual))
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
		space.WriteFile("aaa/bbb/ccc/eee.txt", []byte("CURRENT_CONTENT"))

		generatedTmpl := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `%s
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		testee := factory(mockCtrl, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(
				claude.GenerationResult{
					Content:           fmt.Sprintf(generatedTmpl, "aaa/bbb/ccc/ddd.txt"),
					TerminationReason: "success",
				},
				nil,
			)
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).Return(
				claude.GenerationResult{
					Content:           fmt.Sprintf(generatedTmpl, "aaa/bbb/ccc/eee.txt"),
					TerminationReason: "success",
				},
				nil,
			)
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt", "aaa/bbb/ccc/eee.txt"}, true, false, "", false)
		assert.NoError(t, err)

		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "2022-01-01T00:00:00"))
		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "prompt_01.md"))
		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "answer_01.md"))
		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "prompt_02.md"))
		space.AssertExistPath(filepath.Join(".sisho", "history", "test-ksuid", "answer_02.md"))
	})

	t.Run("Knowledgeスキャンについて", func(t *testing.T) {
		t.Run("相対パスパターン", func(t *testing.T) {
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
			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						// Check if knowledge from .knowledge.yml is included
						assert.Contains(t, content, "aaa/bbb/SPEC.md")
						assert.Contains(t, content, "This is SPEC.md")
						assert.Contains(t, content, "aaa/bbb/SPEC2.md")
						assert.Contains(t, content, "This is SPEC2.md")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
			assert.NoError(t, err)
		})

		t.Run("絶対パスパターン", func(t *testing.T) {
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
			space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

			space.WriteFile("aaa/bbb/SPEC.md", []byte("This is SPEC.md"))
			space.WriteFile("aaa/bbb/ccc/.knowledge.yml", []byte(`
knowledge:
  - path: `+filepath.Join(space.Dir, "aaa/bbb/SPEC.md")+`
    kind: specifications
`))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						// Check if knowledge from .knowledge.yml is included
						assert.Contains(t, content, "aaa/bbb/SPEC.md")
						assert.Contains(t, content, "This is SPEC.md")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
			assert.NoError(t, err)
		})

		t.Run("プロジェクトルートからの相対パスパターン", func(t *testing.T) {
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
			space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

			space.WriteFile("aaa/bbb/SPEC.md", []byte("This is SPEC.md"))
			space.WriteFile("aaa/bbb/ccc/.knowledge.yml", []byte(`
knowledge:
  - path: "@/aaa/bbb/SPEC.md"
    kind: specifications
`))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						// Check if knowledge from .knowledge.yml is included
						assert.Contains(t, content, "aaa/bbb/SPEC.md")
						assert.Contains(t, content, "This is SPEC.md")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
			assert.NoError(t, err)
		})
	})

	t.Run("単一ファイル知識リストファイルについて", func(t *testing.T) {
		t.Run("相対パスパターン", func(t *testing.T) {
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
			space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

			space.WriteFile("aaa/bbb/SPEC.md", []byte("This is SPEC.md"))
			space.WriteFile("aaa/bbb/ccc/ddd.txt.know.yml", []byte(`
knowledge:
  - path: ../SPEC.md
    kind: specifications
`))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						assert.Contains(t, content, "aaa/bbb/SPEC.md")
						assert.Contains(t, content, "This is SPEC.md")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
			assert.NoError(t, err)
		})

		t.Run("絶対パスパターン", func(t *testing.T) {
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
			space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

			space.WriteFile("aaa/bbb/SPEC.md", []byte("This is SPEC.md"))
			space.WriteFile("aaa/bbb/ccc/ddd.txt.know.yml", []byte(`
knowledge:
  - path: `+filepath.Join(space.Dir, "aaa/bbb/SPEC.md")+`
    kind: specifications
`))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						assert.Contains(t, content, "aaa/bbb/SPEC.md")
						assert.Contains(t, content, "This is SPEC.md")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
			assert.NoError(t, err)
		})

		t.Run("プロジェクトルートからの相対パスパターン", func(t *testing.T) {
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
			space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))

			space.WriteFile("aaa/bbb/SPEC.md", []byte("This is SPEC.md"))
			space.WriteFile("aaa/bbb/ccc/ddd.txt.know.yml", []byte(`
knowledge:
  - path: "@/aaa/bbb/SPEC.md"
    kind: specifications
`))

			generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						assert.Contains(t, content, "aaa/bbb/SPEC.md")
						assert.Contains(t, content, "This is SPEC.md")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
			assert.NoError(t, err)
		})
	})

	t.Run("重複する知識ファイルが除外されること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		// Setup Files
		space.WriteFile("sisho.yml", []byte(`
llm:
   driver: anthropic
   model: claude-3-5-sonnet-20240620
auto-collect:
   "[TARGET_CODE].md": true
`))
		space.WriteFile("aaa/bbb/ccc/ddd.txt", []byte("CURRENT_CONTENT"))
		space.WriteFile("aaa/bbb/ccc/ddd.txt.md", []byte("This is ddd.txt.md"))
		space.WriteFile("aaa/.knowledge.yml", []byte(`
knowledge:
  - path: ./bbb/ccc/ddd.txt.md
    kind: specifications
`))
		space.WriteFile("aaa/bbb/.knowledge.yml", []byte(`
knowledge:
  - path: ./ccc/ddd.txt.md
    kind: specifications
`))
		space.WriteFile("aaa/bbb/ccc/ddd.txt.know.yml", []byte(`
knowledge:
  - path: ddd.txt.md
    kind: specifications
`))

		generated := `
<!-- CODE_BLOCK_BEGIN -->` + "```" + `aaa/bbb/ccc/ddd.txt
UPDATED_CONTENT
` + "```" + `<!-- CODE_BLOCK_END -->
`
		testee := factory(mockCtrl, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					content := messages[0].Content
					assert.NotContains(t, content, space.Dir)
					assert.Equal(t, 1, strings.Count(content, "aaa/bbb/ccc/ddd.txt.md"), "aaa/bbb/ccc/ddd.txt.md should be included only once")
					assert.Equal(t, 1, strings.Count(content, "This is ddd.txt.md"), "aaa/bbb/ccc/ddd.txt.md should be included only once")
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
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
		testee := factory(mockCtrl, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					content := messages[0].Content
					assert.NotContains(t, content, space.Dir)
					assert.Contains(t, content, "aaa/bbb/ccc/ddd.txt.md")
					assert.Contains(t, content, "This is ddd.txt.md")
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
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
		testee := factory(mockCtrl, func(mocks Mocks) {
			mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
			mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
				DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
					content := messages[0].Content
					assert.NotContains(t, content, space.Dir)
					assert.NotContains(t, content, "aaa/bbb/ddd.txt.md")
					assert.NotContains(t, content, "This is ddd.txt.md")
					return claude.GenerationResult{
						Content:           generated,
						TerminationReason: "success",
					}, nil
				})
			mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
		})
		err := testee.Make([]string{"aaa/bbb/ccc/ddd.txt"}, true, false, "", false)
		assert.NoError(t, err)
	})

	t.Run("連鎖的生成について", func(t *testing.T) {
		t.Run("連鎖的生成が正常に動作すること", func(t *testing.T) {
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

			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						assert.NotContains(t, messages[0].Content, space.Dir)
						//assert.Contains(t, messages[0].Content, "FILE3_CONTENT")
						return claude.GenerationResult{
							Content:           fmt.Sprintf(generatedFormat, "file3.go", 1),
							TerminationReason: "success",
						}, nil
					})
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						assert.NotContains(t, messages[0].Content, space.Dir)
						//assert.Contains(t, messages[0].Content, "FILE2_CONTENT")
						return claude.GenerationResult{
							Content:           fmt.Sprintf(generatedFormat, "file2.go", 2),
							TerminationReason: "success",
						}, nil
					})
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						assert.NotContains(t, messages[0].Content, space.Dir)
						//assert.Contains(t, messages[0].Content, "FILE1_CONTENT")
						return claude.GenerationResult{
							Content:           fmt.Sprintf(generatedFormat, "file1.go", 3),
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"file3.go"}, true, true, "", false)
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

			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
			})
			err := testee.Make([]string{"file1..go"}, false, true, "", false)

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

			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						assert.NotContains(t, messages[0].Content, space.Dir)
						return claude.GenerationResult{
							Content:           fmt.Sprintf(generatedFormat, "file3.go", 1),
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"file3.go"}, true, true, "", false)
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

			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						assert.Contains(t, content, "# Folder Structure")
						assert.Contains(t, content, "file1.go")
						assert.Contains(t, content, "/dir1")
						assert.Contains(t, content, "file2.go")
						assert.Contains(t, content, "/dir2")
						assert.Contains(t, content, "file3.go")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"file1.go"}, true, false, "", false)
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

			testee := factory(mockCtrl, func(mocks Mocks) {
				mocks.Timer.EXPECT().Now().Return(testUtil.NewTime("2022-01-01T00:00:00Z")).AnyTimes()
				mocks.ClaudeClient.EXPECT().SendMessage(gomock.Any(), gomock.Any()).
					DoAndReturn(func(messages []claude.Message, model string) (claude.GenerationResult, error) {
						content := messages[0].Content
						assert.NotContains(t, content, space.Dir)
						assert.Contains(t, content, "# Folder Structure")
						assert.Contains(t, content, "file1.go")
						assert.Contains(t, content, "/dir1")
						assert.Contains(t, content, "file2.go")
						assert.NotContains(t, content, "ignore_this.txt")
						assert.NotContains(t, content, "/dir2")
						assert.NotContains(t, content, "file3.go")
						return claude.GenerationResult{
							Content:           generated,
							TerminationReason: "success",
						}, nil
					})
				mocks.FileRepository.EXPECT().Getwd().Return(space.Dir, nil).AnyTimes()
				mocks.KsuidGenerator.EXPECT().New().Return("test-ksuid")
			})
			err := testee.Make([]string{"file1.go"}, true, false, "", false)
			assert.NoError(t, err)
		})
	})
}
