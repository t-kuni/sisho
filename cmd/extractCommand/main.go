package extractCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/model/chat"
	modelClaude "github.com/t-kuni/sisho/domain/model/chat/claude"
	modelOpenAi "github.com/t-kuni/sisho/domain/model/chat/openAi"
	"github.com/t-kuni/sisho/domain/model/prompts"
	"github.com/t-kuni/sisho/domain/model/prompts/extract"
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/repository/file"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/folderStructureMake"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"regexp"
)

type ExtractCommand struct {
	CobraCommand *cobra.Command
}

func NewExtractCommand(
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	knowledgeRepository knowledge.Repository,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
) *ExtractCommand {
	cmd := &cobra.Command{
		Use:   "extract [path]",
		Short: "Extract knowledge list from Target Code",
		Long:  `Extract knowledge list from the specified Target Code and generate or update a knowledge list file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtract(args[0], claudeClient, openAiClient, configFindService, configRepository, fileRepository, knowledgeRepository, folderStructureMakeService)
		},
	}

	return &ExtractCommand{
		CobraCommand: cmd,
	}
}

func runExtract(
	path string,
	claudeClient claude.Client,
	openAiClient openAi.Client,
	configFindService *configFindService.ConfigFindService,
	configRepository config.Repository,
	fileRepository file.Repository,
	knowledgeRepository knowledge.Repository,
	folderStructureMakeService *folderStructureMake.FolderStructureMakeService,
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

	targetContent, err := os.ReadFile(path)
	if err != nil {
		return eris.Wrapf(err, "failed to read file: %s", path)
	}

	target := prompts.Target{
		Path:    path,
		Content: string(targetContent),
	}

	folderStructure, err := folderStructureMakeService.MakeTree(rootDir)
	if err != nil {
		return eris.Wrap(err, "failed to get folder structure")
	}

	prompt, err := extract.BuildPrompt(extract.PromptParam{
		Target:          target,
		FolderStructure: folderStructure,
	})
	if err != nil {
		return eris.Wrap(err, "failed to build prompt")
	}

	var chatClient chat.Chat
	switch cfg.LLM.Driver {
	case "open-ai":
		chatClient = modelOpenAi.NewOpenAiChat(openAiClient)
	case "anthropic":
		chatClient = modelClaude.NewClaudeChat(claudeClient)
	default:
		return eris.Errorf("unsupported LLM driver: %s", cfg.LLM.Driver)
	}

	answer, err := chatClient.Send(prompt, cfg.LLM.Model)
	if err != nil {
		return eris.Wrap(err, "failed to send message to LLM")
	}

	knowledgeList, err := extractKnowledgeList(answer, path)
	if err != nil {
		return eris.Wrap(err, "failed to extract knowledge list")
	}

	knowledgeFilePath := getKnowledgeListFilePath(path)
	existingKnowledgeFile, err := knowledgeRepository.Read(knowledgeFilePath)
	if err == nil {
		knowledgeList = mergeKnowledgeLists(existingKnowledgeFile.KnowledgeList, knowledgeList)
	}

	err = knowledgeRepository.Write(knowledgeFilePath, knowledge.KnowledgeFile{KnowledgeList: knowledgeList})
	if err != nil {
		return eris.Wrapf(err, "failed to write knowledge file: %s", knowledgeFilePath)
	}

	fmt.Printf("Knowledge list extracted and saved to: %s\n", knowledgeFilePath)
	return nil
}

func extractKnowledgeList(answer string, path string) ([]knowledge.Knowledge, error) {
	re := regexp.MustCompile("(?s)(\n|^)<!-- CODE_BLOCK_BEGIN -->```" + regexp.QuoteMeta(getKnowledgeListFilePath(path)) + "(.*)```.?<!-- CODE_BLOCK_END -->(\n|$)")
	matches := re.FindStringSubmatch(answer)

	if len(matches) < 3 {
		return nil, eris.New("no knowledge list found in the answer")
	}

	yamlContent := matches[2]

	var knowledgeFile knowledge.KnowledgeFile
	err := yaml.Unmarshal([]byte(yamlContent), &knowledgeFile)
	if err != nil {
		return nil, eris.Wrap(err, "failed to unmarshal YAML content")
	}

	for i, k := range knowledgeFile.KnowledgeList {
		relPath, err := filepath.Rel(filepath.Dir(path), k.Path)
		if err != nil {
			return nil, eris.Wrapf(err, "failed to get relative path for: %s", k.Path)
		}
		knowledgeFile.KnowledgeList[i].Path = relPath
	}

	return knowledgeFile.KnowledgeList, nil
}

func getKnowledgeListFilePath(path string) string {
	dir := filepath.Dir(path)
	fileName := filepath.Base(path)
	return filepath.Join(dir, fileName+".know.yml")
}

func mergeKnowledgeLists(existing, new []knowledge.Knowledge) []knowledge.Knowledge {
	merged := make([]knowledge.Knowledge, 0)
	seen := make(map[string]bool)

	for _, k := range existing {
		if !seen[k.Path] {
			merged = append(merged, k)
			seen[k.Path] = true
		}
	}

	for _, k := range new {
		if !seen[k.Path] {
			merged = append(merged, k)
			seen[k.Path] = true
		}
	}

	return merged
}
