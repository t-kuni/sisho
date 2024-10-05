package makeCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	chat2 "github.com/t-kuni/sisho/domain/model/chat"
	modelClaude "github.com/t-kuni/sisho/domain/model/chat/claude"
	modelOpenAi "github.com/t-kuni/sisho/domain/model/chat/openAi"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"github.com/t-kuni/sisho/domain/model/prompts/oneMoreMake"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/depsGraph"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// MakeCommand は、makeコマンドの構造体です。
type MakeCommand struct {
	CobraCommand   *cobra.Command
	claudeClient   claude.Client
	openAiClient   openAi.Client
	timer          timer.ITimer
	ksuidGenerator ksuid.IKsuid
}

// NewMakeCommand は、MakeCommandの新しいインスタンスを作成します。
func NewMakeCommand(
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	knowledgeScanService *knowledgeScan.KnowledgeScanService,
	knowledgeLoadService *knowledgeLoad.KnowledgeLoadService,
	depsGraphRepo depsGraph.Repository,
	timer timer.ITimer,
	ksuidGenerator ksuid.IKsuid,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
) *MakeCommand {
	var promptFlag bool
	var applyFlag bool
	var chainFlag bool

	cmd := &cobra.Command{
		Use:   "make [path...]",
		Short: "Generate files using LLM",
		Long:  `Generate files at the specified paths using LLM based on the knowledge sets.`,
		Args:  cobra.MinimumNArgs(1),
		RunE: runMake(&promptFlag, &applyFlag, &chainFlag, claudeClient, openAiClient, configFindService, configRepository,
			fileRepository, knowledgeScanService, knowledgeLoadService, depsGraphRepo, timer, ksuidGenerator, folderStructureMakeService),
	}

	cmd.Flags().BoolVarP(&promptFlag, "prompt", "p", false, "Open editor for additional instructions")
	cmd.Flags().BoolVarP(&applyFlag, "apply", "a", false, "Apply LLM output to files")
	cmd.Flags().BoolVarP(&chainFlag, "chain", "c", false, "Include dependent files based on deps-graph")

	return &MakeCommand{
		CobraCommand:   cmd,
		claudeClient:   claudeClient,
		openAiClient:   openAiClient,
		timer:          timer,
		ksuidGenerator: ksuidGenerator,
	}
}

// runMake は、makeコマンドの主要なロジックを実行します。
func runMake(
	promptFlag *bool,
	applyFlag *bool,
	chainFlag *bool,
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	knowledgeScanService *knowledgeScan.KnowledgeScanService,
	knowledgeLoadService *knowledgeLoad.KnowledgeLoadService,
	depsGraphRepo depsGraph.Repository,
	timer timer.ITimer,
	ksuidGenerator ksuid.IKsuid,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// 設定ファイルの読み込み
		configPath, err := configFindService.FindConfig()
		if err != nil {
			return eris.Wrap(err, "failed to find config file")
		}

		cfg, err := configRepository.Read(configPath)
		if err != nil {
			return eris.Wrap(err, "failed to read config file")
		}

		rootDir := configFindService.GetProjectRoot(configPath)

		// チェーンフラグが設定されている場合、依存グラフを使用してターゲットを拡張
		if *chainFlag {
			args, err = expandTargetsWithDependencies(args, rootDir, depsGraphRepo)
			if err != nil {
				if os.IsNotExist(err) {
					return eris.Wrap(err, "deps-graph.json not found")
				}
				return eris.Wrap(err, "failed to expand targets with dependencies")
			}
		}

		// Target Codeの一覧を標準出力に出力
		fmt.Println("Target Codes:")
		for _, arg := range args {
			fmt.Printf("- %s\n", arg)
		}
		fmt.Println()

		// 知識のスキャンとロード
		scannedKnowledge, err := knowledgeScanService.ScanKnowledge(rootDir, args)
		if err != nil {
			return eris.Wrap(err, "failed to scan knowledge")
		}

		knowledgeSets, err := knowledgeLoadService.LoadKnowledge(rootDir, scannedKnowledge)
		if err != nil {
			return eris.Wrap(err, "failed to load knowledge")
		}

		// 追加の指示の取得
		var instructions string
		if *promptFlag {
			instructions, err = getAdditionalInstructions()
			if err != nil {
				return eris.Wrap(err, "failed to get additional instructions")
			}
			fmt.Println("Additional instructions:")
			fmt.Println(instructions)
		} else {
			instructions, err = readStdin()
			if err != nil {
				return eris.Wrap(err, "failed to read from stdin")
			}
		}

		printKnowledgePaths(knowledgeSets)

		// チャットモデルの選択
		var chat chat2.Chat
		switch cfg.LLM.Driver {
		case "open-ai":
			chat = modelOpenAi.NewOpenAiChat(openAiClient)
		case "anthropic":
			chat = modelClaude.NewClaudeChat(claudeClient)
		default:
			return eris.Errorf("unsupported LLM driver: %s", cfg.LLM.Driver)
		}

		fmt.Printf("Using LLM: %s with model: %s\n", cfg.LLM.Driver, cfg.LLM.Model)

		// 履歴ディレクトリの作成
		historyDir, err := createHistoryDir(rootDir, timer, ksuidGenerator)
		if err != nil {
			return eris.Wrap(err, "failed to create history directory")
		}

		// 各ターゲットに対する処理
		for i, path := range args {
			target, err := readTarget(path, fileRepository)
			if err != nil {
				return eris.Wrapf(err, "failed to read target: %s", path)
			}

			var prompt string
			if i == 0 {
				folderStructure := ""
				if cfg.AdditionalKnowledge.FolderStructure {
					folderStructure, err = folderStructureMakeService.MakeTree(rootDir)
					if err != nil {
						return eris.Wrap(err, "failed to get folder structure")
					}
				}

				prompt, err = prompts.BuildPrompt(prompts.PromptParam{
					KnowledgeSets:   knowledgeSets,
					Targets:         []prompts.Target{target},
					Instructions:    instructions,
					FolderStructure: folderStructure,
				})
			} else {
				prompt, err = oneMoreMake.BuildPrompt(oneMoreMake.PromptParam{
					Path: path,
				})
			}
			if err != nil {
				return eris.Wrap(err, "failed to build prompt")
			}

			answer, err := chat.Send(prompt, cfg.LLM.Model)
			if err != nil {
				return eris.Wrap(err, "failed to send message to LLM")
			}

			err = saveHistory(historyDir, prompt, answer)
			if err != nil {
				return eris.Wrap(err, "failed to save history")
			}

			if *applyFlag {
				err = applyChanges(path, answer, fileRepository)
				if err != nil {
					return eris.Wrapf(err, "failed to apply changes to %s", path)
				}
				fmt.Printf("Applied changes to %s\n", path)
			} else {
				fmt.Println(answer)
			}
		}

		return nil
	}
}

// expandTargetsWithDependencies は、依存グラフを使用してターゲットを拡張します。
func expandTargetsWithDependencies(targets []string, rootDir string, depsGraphRepo depsGraph.Repository) ([]string, error) {
	graph, err := depsGraphRepo.Read(filepath.Join(rootDir, ".sisho", "deps-graph.json"))
	if err != nil {
		return nil, eris.Wrap(err, "failed to read deps-graph.json")
	}

	expandedTargets := make(map[string]struct{})
	for _, target := range targets {
		expandDependencies(target, graph, expandedTargets)
	}

	result := make([]string, 0, len(expandedTargets))
	for target := range expandedTargets {
		result = append(result, target)
	}

	// 依存グラフに基づいてソート
	sort.Slice(result, func(i, j int) bool {
		return getDepth(graph, result[i]) > getDepth(graph, result[j])
	})

	return result, nil
}

// expandDependencies は、指定されたターゲットの依存関係を再帰的に展開します。
func expandDependencies(target string, graph depsGraph.DepsGraph, expandedTargets map[string]struct{}) {
	if _, exists := expandedTargets[target]; exists {
		return
	}
	expandedTargets[target] = struct{}{}

	deps, exists := graph[depsGraph.Dependency(target)]
	if !exists {
		// ターゲットが依存グラフに存在しない場合は、最も深い深度として扱う
		return
	}
	for _, dep := range deps {
		expandDependencies(string(dep), graph, expandedTargets)
	}
}

// getAdditionalInstructions は、ユーザーから追加の指示を取得します。
func getAdditionalInstructions() (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	tempFile, err := os.CreateTemp("", "sisho-instructions-*.md")
	if err != nil {
		return "", eris.Wrap(err, "failed to create temporary file")
	}
	defer os.Remove(tempFile.Name())

	cmd := exec.Command(editor, tempFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", eris.Wrap(err, "failed to run editor")
	}

	instructions, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return "", eris.Wrap(err, "failed to read instructions from temporary file")
	}

	return strings.TrimSpace(string(instructions)), nil
}

// readStdin は標準入力からテキストを読み取ります。
func readStdin() (string, error) {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", eris.Wrap(err, "failed to read from stdin")
	}
	return strings.TrimSpace(string(stdin)), nil
}

// printKnowledgePaths は、ナレッジのパスを出力します。
func printKnowledgePaths(knowledgeSets []prompts.KnowledgeSet) {
	fmt.Println("Knowledge paths:")
	for _, set := range knowledgeSets {
		for _, k := range set.Knowledge {
			fmt.Printf("- %s (%s)\n", k.Path, set.Kind)
		}
	}
	fmt.Println()
}

// readTarget は、指定されたパスのターゲットを読み込みます。
func readTarget(path string, fileRepository file.Repository) (prompts.Target, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return prompts.Target{}, eris.Wrapf(err, "failed to read file: %s", path)
		}
		content = []byte{}
	}

	return prompts.Target{
		Path:    path,
		Content: string(content),
	}, nil
}

// createHistoryDir は、履歴ディレクトリを作成します。
func createHistoryDir(rootDir string, timer timer.ITimer, ksuidGenerator ksuid.IKsuid) (string, error) {
	historyBaseDir := filepath.Join(rootDir, ".sisho", "history")
	err := os.MkdirAll(historyBaseDir, 0755)
	if err != nil {
		return "", eris.Wrap(err, "failed to create history base directory")
	}

	id := ksuidGenerator.New()
	historyDir := filepath.Join(historyBaseDir, id)
	err = os.Mkdir(historyDir, 0755)
	if err != nil {
		return "", eris.Wrap(err, "failed to create history directory")
	}

	timeFile := filepath.Join(historyDir, timer.Now().Format("2006-01-02T15:04:05"))
	_, err = os.Create(timeFile)
	if err != nil {
		return "", eris.Wrap(err, "failed to create time file")
	}

	return historyDir, nil
}

// saveHistory は、プロンプトと回答を履歴として保存します。
func saveHistory(historyDir, prompt, answer string) error {
	err := os.WriteFile(filepath.Join(historyDir, "prompt.md"), []byte(prompt), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write prompt to history")
	}

	err = os.WriteFile(filepath.Join(historyDir, "answer.md"), []byte(answer), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write answer to history")
	}

	return nil
}

// applyChanges は、LLMの出力をファイルに適用します。
func applyChanges(path, answer string, fileRepository file.Repository) error {
	// この正規表現は絶対に変更しないでください
	// gpt-4だとコードブロックの終了からコメントの間に1文字入ることがある
	re := regexp.MustCompile("(?s)(\n|^)<!-- CODE_BLOCK_BEGIN -->```" + regexp.QuoteMeta(path) + "(.*)```.?<!-- CODE_BLOCK_END -->(\n|$)")
	matches := re.FindStringSubmatch(answer)

	if len(matches) < 4 {
		return eris.New("no code block found in the answer")
	}

	newContent := strings.TrimSpace(matches[2])

	oldContent, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return eris.Wrapf(err, "failed to read file: %s", path)
	}

	if string(oldContent) != newContent {
		err = write(path, []byte(newContent))
		if err != nil {
			return eris.Wrapf(err, "failed to write file: %s", path)
		}

		printDiff(string(oldContent), newContent)
	}

	return nil
}

// printDiff は、古い内容と新しい内容の差分を出力します。
func printDiff(oldContent, newContent string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)
	fmt.Println(dmp.DiffPrettyText(diffs))
}

// write は、指定されたパスにデータを書き込みます。
func write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return eris.Wrapf(err, "failed to create directory: %s", dir)
	}
	return eris.Wrapf(os.WriteFile(path, data, 0644), "failed to write file: %s", path)
}

// getDepth は、依存グラフにおける指定されたノードの深さを返します。
func getDepth(g depsGraph.DepsGraph, node string) int {
	visited := make(map[string]bool)
	var dfs func(string) int
	dfs = func(current string) int {
		if visited[current] {
			return 0
		}
		visited[current] = true
		maxDepth := 0
		deps, exists := g[depsGraph.Dependency(current)]
		if !exists {
			// ノードが依存グラフに存在しない場合は、最も深い深度として扱う
			return 999999 // 十分に大きな値
		}
		for _, dep := range deps {
			depDepth := dfs(string(dep))
			if depDepth > maxDepth {
				maxDepth = depDepth
			}
		}
		return maxDepth + 1
	}
	return dfs(node)
}
