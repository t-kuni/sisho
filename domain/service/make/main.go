package make

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	chat2 "github.com/t-kuni/sisho/domain/model/chat"
	modelClaude "github.com/t-kuni/sisho/domain/model/chat/claude"
	modelOpenAi "github.com/t-kuni/sisho/domain/model/chat/openAi"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/depsGraph"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgeLoad"
	"github.com/t-kuni/sisho/domain/service/knowledgeScan"
	"github.com/t-kuni/sisho/domain/system/ksuid"
	"github.com/t-kuni/sisho/domain/system/timer"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type MakeService struct {
	claudeClient               claude.Client
	openAiClient               openAi.Client
	configFindService          *configFindService.ConfigFindService
	configRepository           config.Repository
	fileRepository             file.Repository
	knowledgeScanService       *knowledgeScan.KnowledgeScanService
	knowledgeLoadService       *knowledgeLoad.KnowledgeLoadService
	depsGraphRepo              depsGraph.Repository
	timer                      timer.ITimer
	ksuidGenerator             ksuid.IKsuid
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService
}

func NewMakeService(
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
) *MakeService {
	return &MakeService{
		claudeClient:               claudeClient,
		openAiClient:               openAiClient,
		configFindService:          configFindService,
		configRepository:           configRepository,
		fileRepository:             fileRepository,
		knowledgeScanService:       knowledgeScanService,
		knowledgeLoadService:       knowledgeLoadService,
		depsGraphRepo:              depsGraphRepo,
		timer:                      timer,
		ksuidGenerator:             ksuidGenerator,
		folderStructureMakeService: folderStructureMakeService,
	}
}

func (s *MakeService) Make(paths []string, applyFlag, chainFlag bool, instructions string) error {
	// 設定ファイルの読み込み
	configPath, err := s.configFindService.FindConfig()
	if err != nil {
		return eris.Wrap(err, "failed to find config file")
	}

	cfg, err := s.configRepository.Read(configPath)
	if err != nil {
		return eris.Wrap(err, "failed to read config file")
	}

	rootDir := s.configFindService.GetProjectRoot(configPath)

	// チェーンフラグが設定されている場合、依存グラフを使用してターゲットを拡張
	if chainFlag {
		paths, err = s.expandTargetsWithDependencies(paths, rootDir)
		if err != nil {
			if os.IsNotExist(err) {
				return eris.Wrap(err, "deps-graph.json not found")
			}
			return eris.Wrap(err, "failed to expand targets with dependencies")
		}
	}

	// Target Codeの一覧を標準出力に出力
	fmt.Println("Target Codes:")
	for _, arg := range paths {
		fmt.Printf("- %s\n", arg)
	}
	fmt.Println()

	fmt.Printf("Using LLM: %s with model: %s\n", cfg.LLM.Driver, cfg.LLM.Model)

	// 履歴ディレクトリの作成
	historyDir, err := s.createHistoryDir(rootDir)
	if err != nil {
		return eris.Wrap(err, "failed to create history directory")
	}

	// フォルダ構造情報の取得
	var folderStructure string
	if cfg.AdditionalKnowledge.FolderStructure {
		folderStructure, err = s.folderStructureMakeService.MakeTree(rootDir)
		if err != nil {
			return eris.Wrap(err, "failed to get folder structure")
		}
	}

	// 各ターゲットに対する処理
	for i, path := range paths {
		fmt.Printf("\n--- Processing target: %s ---\n", path)

		// チャットモデルの選択
		var chat chat2.Chat
		switch cfg.LLM.Driver {
		case "open-ai":
			chat = modelOpenAi.NewOpenAiChat(s.openAiClient)
		case "anthropic":
			chat = modelClaude.NewClaudeChat(s.claudeClient)
		default:
			return eris.Errorf("unsupported LLM driver: %s", cfg.LLM.Driver)
		}

		// Target Codeの読み込み
		targets, err := s.readAllTargets(paths)
		if err != nil {
			return eris.Wrap(err, "failed to read all targets")
		}

		// 知識のスキャンとロード
		scannedKnowledge, err := s.knowledgeScanService.ScanKnowledgeMultipleTarget(rootDir, paths)
		if err != nil {
			return eris.Wrap(err, "failed to scan knowledge")
		}

		knowledgeSets, err := s.knowledgeLoadService.LoadKnowledge(rootDir, scannedKnowledge)
		if err != nil {
			return eris.Wrap(err, "failed to load knowledge")
		}

		s.printKnowledgePaths(knowledgeSets)

		// プロンプトの生成
		prompt, err := prompts.BuildPrompt(prompts.PromptParam{
			KnowledgeSets:   knowledgeSets,
			Targets:         targets,
			Instructions:    instructions,
			FolderStructure: folderStructure,
			GeneratePath:    path,
		})
		if err != nil {
			return eris.Wrap(err, "failed to build prompt")
		}

		err = s.savePromptHistory(historyDir, i+1, prompt)
		if err != nil {
			return eris.Wrap(err, "failed to save prompt history")
		}

		answer, err := chat.Send(prompt, cfg.LLM.Model)
		if err != nil {
			return eris.Wrap(err, "failed to send message to LLM")
		}

		err = s.saveAnswerHistory(historyDir, i+1, answer)
		if err != nil {
			return eris.Wrap(err, "failed to save answer history")
		}

		if applyFlag {
			err = s.applyChanges(path, answer)
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

func (s *MakeService) expandTargetsWithDependencies(targets []string, rootDir string) ([]string, error) {
	graph, err := s.depsGraphRepo.Read(filepath.Join(rootDir, ".sisho", "deps-graph.json"))
	if err != nil {
		return nil, eris.Wrap(err, "failed to read deps-graph.json")
	}

	expandedTargets := make(map[string]struct{})
	for _, target := range targets {
		s.expandDependencies(target, graph, expandedTargets)
	}

	result := make([]string, 0, len(expandedTargets))
	for target := range expandedTargets {
		result = append(result, target)
	}

	// 依存グラフに基づいてソート
	sort.Slice(result, func(i, j int) bool {
		return s.getDepth(graph, result[i]) > s.getDepth(graph, result[j])
	})

	return result, nil
}

func (s *MakeService) expandDependencies(target string, graph depsGraph.DepsGraph, expandedTargets map[string]struct{}) {
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
		s.expandDependencies(string(dep), graph, expandedTargets)
	}
}

func (s *MakeService) printKnowledgePaths(knowledgeSets []prompts.KnowledgeSet) {
	fmt.Println("Knowledge paths:")
	for _, set := range knowledgeSets {
		for _, k := range set.Knowledge {
			fmt.Printf("- %s (%s)\n", k.Path, set.Kind)
		}
	}
	fmt.Println()
}

func (s *MakeService) readAllTargets(paths []string) ([]prompts.Target, error) {
	targets := make([]prompts.Target, len(paths))
	for i, path := range paths {
		target, err := s.readTarget(path)
		if err != nil {
			return nil, eris.Wrapf(err, "failed to read target: %s", path)
		}
		targets[i] = target
	}
	return targets, nil
}

func (s *MakeService) readTarget(path string) (prompts.Target, error) {
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

func (s *MakeService) createHistoryDir(rootDir string) (string, error) {
	historyBaseDir := filepath.Join(rootDir, ".sisho", "history")
	err := os.MkdirAll(historyBaseDir, 0755)
	if err != nil {
		return "", eris.Wrap(err, "failed to create history base directory")
	}

	id := s.ksuidGenerator.New()
	historyDir := filepath.Join(historyBaseDir, id)
	err = os.Mkdir(historyDir, 0755)
	if err != nil {
		return "", eris.Wrap(err, "failed to create history directory")
	}

	timeFile := filepath.Join(historyDir, s.timer.Now().Format("2006-01-02T15:04:05"))
	_, err = os.Create(timeFile)
	if err != nil {
		return "", eris.Wrap(err, "failed to create time file")
	}

	return historyDir, nil
}

func (s *MakeService) savePromptHistory(historyDir string, index int, prompt string) error {
	filename := fmt.Sprintf("prompt_%02d.md", index)
	err := os.WriteFile(filepath.Join(historyDir, filename), []byte(prompt), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write prompt to history")
	}
	return nil
}

func (s *MakeService) saveAnswerHistory(historyDir string, index int, answer string) error {
	filename := fmt.Sprintf("answer_%02d.md", index)
	err := os.WriteFile(filepath.Join(historyDir, filename), []byte(answer), 0644)
	if err != nil {
		return eris.Wrap(err, "failed to write answer to history")
	}
	return nil
}

func (s *MakeService) applyChanges(path, answer string) error {
	// NOTE: この正規表現は絶対に変更しないでください
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
		err = s.write(path, []byte(newContent))
		if err != nil {
			return eris.Wrapf(err, "failed to write file: %s", path)
		}

		s.printDiff(string(oldContent), newContent)
	}

	return nil
}

func (s *MakeService) printDiff(oldContent, newContent string) {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldContent, newContent, false)
	fmt.Println(dmp.DiffPrettyText(diffs))
}

func (s *MakeService) write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return eris.Wrapf(err, "failed to create directory: %s", dir)
	}
	return eris.Wrapf(os.WriteFile(path, data, 0644), "failed to write file: %s", path)
}

func (s *MakeService) getDepth(g depsGraph.DepsGraph, node string) int {
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
