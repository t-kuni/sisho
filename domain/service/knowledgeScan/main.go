package knowledgeScan

import (
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"path/filepath"
)

type KnowledgeScanService struct {
	knowledgeRepo      knowledge.Repository
	autoCollectService *autoCollect.AutoCollectService
}

func NewKnowledgeScanService(knowledgeRepo knowledge.Repository, autoCollectService *autoCollect.AutoCollectService) *KnowledgeScanService {
	return &KnowledgeScanService{
		knowledgeRepo:      knowledgeRepo,
		autoCollectService: autoCollectService,
	}
}

func (s *KnowledgeScanService) ScanKnowledge(rootDir string, targetPaths []string) ([]knowledge.Knowledge, error) {
	uniqueKnowledge := make(map[string]knowledge.Knowledge)

	for _, targetPath := range targetPaths {
		// .knowledge.ymlファイルのスキャン
		knowledgeFromYml, err := s.scanKnowledgeYml(rootDir, targetPath)
		if err != nil {
			return nil, err
		}
		for _, k := range knowledgeFromYml {
			uniqueKnowledge[k.Path] = k
		}

		// auto-collectの処理
		autoCollectedFiles, err := s.autoCollectService.CollectAutoCollectFiles(rootDir, targetPath)
		if err != nil {
			return nil, err
		}
		for _, file := range autoCollectedFiles {
			uniqueKnowledge[file] = knowledge.Knowledge{
				Path: file,
				Kind: "specifications", // auto-collectされたファイルはspecificationsとして扱う
			}
		}
	}

	var knowledgeList []knowledge.Knowledge
	for _, k := range uniqueKnowledge {
		knowledgeList = append(knowledgeList, k)
	}

	return knowledgeList, nil
}

func (s *KnowledgeScanService) scanKnowledgeYml(rootDir string, targetPath string) ([]knowledge.Knowledge, error) {
	var knowledgeList []knowledge.Knowledge
	currentDir := filepath.Dir(targetPath)

	for {
		knowledgeFilePath := filepath.Join(currentDir, ".knowledge.yml")
		knowledgeFile, err := s.knowledgeRepo.Read(knowledgeFilePath)
		if err == nil {
			for _, k := range knowledgeFile.KnowledgeList {
				relPath, err := filepath.Rel(rootDir, filepath.Join(currentDir, k.Path))
				if err != nil {
					return nil, err
				}
				k.Path = relPath
				knowledgeList = append(knowledgeList, k)
			}
		}

		if currentDir == "." {
			break
		}
		currentDir = filepath.Dir(currentDir)
	}

	return knowledgeList, nil
}
