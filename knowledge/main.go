package knowledge

import (
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/prompts"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type KnowledgeFile struct {
	KnowledgeList []Knowledge `yaml:"knowledge"`
}

type Knowledge struct {
	Path string `yaml:"path"`
	Kind string `yaml:"kind"`
}

func ConvertToKnowledgeSet(rootDir string, knowledgeList []Knowledge) ([]prompts.KnowledgeSet, error) {
	kindMap := make(map[string][]prompts.Knowledge)

	for _, knowledge := range knowledgeList {
		content, err := os.ReadFile(filepath.Join(rootDir, knowledge.Path))
		if err != nil {
			return nil, err
		}

		converted := prompts.Knowledge{
			Path:    knowledge.Path,
			Content: string(content),
		}
		kindMap[knowledge.Kind] = append(kindMap[knowledge.Kind], converted)
	}

	var knowledgeSets []prompts.KnowledgeSet
	for kind, knowledges := range kindMap {
		knowledgeSets = append(knowledgeSets, prompts.KnowledgeSet{
			Kind:      kind,
			Knowledge: knowledges,
		})
	}

	return knowledgeSets, nil
}

func ScanKnowledge(rootDir string, targetPaths []string, autoCollectService *autoCollect.AutoCollectService) ([]Knowledge, error) {
	uniqueKnowledge := make(map[string]Knowledge)

	for _, targetPath := range targetPaths {
		// .knowledge.ymlファイルのスキャン
		knowledgeFromYml, err := scanKnowledgeYml(rootDir, targetPath)
		if err != nil {
			return nil, err
		}
		for _, k := range knowledgeFromYml {
			uniqueKnowledge[k.Path] = k
		}

		// auto-collectの処理
		autoCollectedFiles, err := autoCollectService.CollectAutoCollectFiles(rootDir, targetPath)
		if err != nil {
			return nil, err
		}
		for _, file := range autoCollectedFiles {
			uniqueKnowledge[file] = Knowledge{
				Path: file,
				Kind: "specifications", // auto-collectされたファイルはspecificationsとして扱う
			}
		}
	}

	var knowledgeList []Knowledge
	for _, k := range uniqueKnowledge {
		knowledgeList = append(knowledgeList, k)
	}

	return knowledgeList, nil
}

func scanKnowledgeYml(rootDir string, targetPath string) ([]Knowledge, error) {
	var knowledgeList []Knowledge
	currentDir, err := filepath.Abs(filepath.Dir(targetPath))
	if err != nil {
		return nil, err
	}

	for {
		knowledgeFilePath := filepath.Join(currentDir, ".knowledge.yml")
		if _, err := os.Stat(knowledgeFilePath); !os.IsNotExist(err) {
			f, err := ReadKnowledge(knowledgeFilePath)
			if err != nil {
				return nil, err
			}
			for _, k := range f.KnowledgeList {
				relPath, err := filepath.Rel(rootDir, filepath.Join(currentDir, k.Path))
				if err != nil {
					return nil, err
				}
				k.Path = relPath
				knowledgeList = append(knowledgeList, k)
			}
		}

		if currentDir == rootDir {
			break
		}
		currentDir = filepath.Dir(currentDir)
	}

	return knowledgeList, nil
}

func ReadKnowledge(path string) (KnowledgeFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return KnowledgeFile{}, err
	}

	var k KnowledgeFile
	err = yaml.Unmarshal(content, &k)
	if err != nil {
		return KnowledgeFile{}, err
	}

	return k, nil
}

func WriteKnowledge(path string, k KnowledgeFile) error {
	content, err := yaml.Marshal(k)
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}
