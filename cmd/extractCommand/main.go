package extractCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"github.com/t-kuni/sisho/domain/model/prompts/extract"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/chatFactory"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/extractCodeBlock"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type ExtractCommand struct {
	CobraCommand *cobra.Command
}

func NewExtractCommand(
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	knowledgeRepository knowledge.Repository,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
	knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService,
	extractCodeBlockService *extractCodeBlock.CodeBlockExtractService,
	chatFactory *chatFactory.ChatFactory,
) *ExtractCommand {
	cmd := &cobra.Command{
		Use:   "extract [path]",
		Short: "Extract knowledge list from Target Code",
		Long:  `Extract knowledge list from the specified Target Code and generate or update a knowledge list file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtract(args[0], configFindService, configRepository, knowledgeRepository,
				folderStructureMakeService, knowledgePathNormalizeService, extractCodeBlockService, chatFactory)
		},
	}

	return &ExtractCommand{
		CobraCommand: cmd,
	}
}

func runExtract(
	path string,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	knowledgeRepository knowledge.Repository,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
	knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService,
	extractCodeBlockService *extractCodeBlock.CodeBlockExtractService,
	chatFactory *chatFactory.ChatFactory,
) error {
	configPath, err := configFindService.FindConfig()
	if err != nil {
		return eris.Wrap(err, "failed to find config file")
	}

	cfg, err := configRepository.Read(configPath)
	if err != nil {
		return eris.Wrap(err, "failed to read config file")
	}

	rootDir := configFindService.GetProjectRoot(configPath)

	// Convert path to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return eris.Wrapf(err, "failed to get absolute path for: %s", path)
	}

	targetContent, err := os.ReadFile(absPath)
	if err != nil {
		return eris.Wrapf(err, "failed to read file: %s", absPath)
	}

	relPath, err := filepath.Rel(rootDir, absPath)
	if err != nil {
		return eris.Wrapf(err, "failed to get relative path for: %s", absPath)
	}

	target := prompts.Target{
		Path:    relPath,
		Content: string(targetContent),
	}

	folderStructure, err := folderStructureMakeService.MakeTree(rootDir)
	if err != nil {
		return eris.Wrap(err, "failed to get folder structure")
	}

	knowledgeListPath := getKnowledgeListFilePath(absPath)
	relativeKnowledgeListPath, err := filepath.Rel(rootDir, knowledgeListPath)
	if err != nil {
		return eris.Wrap(err, "failed to get relative knowledge list path")
	}

	prompt, err := extract.BuildPrompt(extract.PromptParam{
		Target:            target,
		FolderStructure:   folderStructure,
		KnowledgeListPath: relativeKnowledgeListPath,
	})
	if err != nil {
		return eris.Wrap(err, "failed to build prompt")
	}

	chatClient, err := chatFactory.Make(cfg)
	if err != nil {
		return eris.Wrap(err, "failed to create chat client")
	}

	answer, err := chatClient.Send(prompt, cfg.LLM.Model)
	if err != nil {
		return eris.Wrap(err, "failed to send message to LLM")
	}

	knowledgeList, err := extractKnowledgeList(extractCodeBlockService, answer.Content, relativeKnowledgeListPath, rootDir, knowledgePathNormalizeService)
	if err != nil {
		return eris.Wrap(err, "failed to extract knowledge list")
	}

	existingKnowledgeFile := knowledge.KnowledgeFile{
		KnowledgeList: []knowledge.Knowledge{},
	}
	if _, err := os.Stat(knowledgeListPath); err == nil {
		existingKnowledgeFile, err = knowledgeRepository.Read(knowledgeListPath)
		if err != nil {
			return eris.Wrapf(err, "failed to read existing knowledge file: %s", knowledgeListPath)
		}
	}
	knowledgeList = mergeKnowledgeLists(existingKnowledgeFile.KnowledgeList, knowledgeList, knowledgePathNormalizeService, rootDir, knowledgeListPath)

	err = knowledgeRepository.Write(knowledgeListPath, knowledge.KnowledgeFile{KnowledgeList: knowledgeList})
	if err != nil {
		return eris.Wrapf(err, "failed to write knowledge file: %s", knowledgeListPath)
	}

	fmt.Printf("Knowledge list extracted and saved to: %s\n", knowledgeListPath)
	return nil
}

func extractKnowledgeList(extractCodeBlockService *extractCodeBlock.CodeBlockExtractService, answer string, knowledgeListPath string, rootDir string, knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService) ([]knowledge.Knowledge, error) {
	yamlContent, err := extractCodeBlockService.ExtractCodeBlock(answer, knowledgeListPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to extract code block")
	}

	var knowledgeFile knowledge.KnowledgeFile
	err = yaml.Unmarshal([]byte(yamlContent), &knowledgeFile)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal YAML content")
	}

	return knowledgeFile.KnowledgeList, nil
}

func getKnowledgeListFilePath(path string) string {
	dir := filepath.Dir(path)
	fileName := filepath.Base(path)
	return filepath.Join(dir, fileName+".know.yml")
}

func mergeKnowledgeLists(existing, new []knowledge.Knowledge, knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService, rootDir, knowledgeListPath string) []knowledge.Knowledge {
	merged := make([]knowledge.Knowledge, 0)
	seen := make(map[string]bool)

	for _, k := range existing {
		normalizedPath, _ := knowledgePathNormalizeService.NormalizePath(rootDir, filepath.Dir(knowledgeListPath), k.Path)
		if !seen[normalizedPath] {
			merged = append(merged, k)
			seen[normalizedPath] = true
		}
	}

	for _, k := range new {
		normalizedPath, _ := knowledgePathNormalizeService.NormalizePath(rootDir, filepath.Dir(knowledgeListPath), k.Path)
		if !seen[normalizedPath] {
			// LLMの回答から抽出した知識リストのパスは@表記の相対パスに直す
			k.Path = "@/" + filepath.Clean(k.Path)
			merged = append(merged, k)
			seen[normalizedPath] = true
		}
	}

	return merged
}
