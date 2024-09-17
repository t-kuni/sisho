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
		// レイヤー知識リストファイル（.knowledge.yml）のスキャン
		knowledgeFromYml, err := s.scanKnowledgeYml(rootDir, targetPath)
		if err != nil {
			return nil, err
		}
		for _, k := range knowledgeFromYml {
			uniqueKnowledge[k.Path] = k
		}

		// 単一ファイル知識リストファイル（[ファイル名].know.yml）のスキャン
		knowledgeFromKnowYml, err := s.scanKnowledgeKnowYml(rootDir, targetPath)
		if err != nil {
			return nil, err
		}
		for _, k := range knowledgeFromKnowYml {
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
				k.Path = filepath.Clean(filepath.Join(currentDir, k.Path))
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

func (s *KnowledgeScanService) scanKnowledgeKnowYml(rootDir string, targetPath string) ([]knowledge.Knowledge, error) {
	var knowledgeList []knowledge.Knowledge

	fileName := filepath.Base(targetPath)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	knowYmlPath := filepath.Join(filepath.Dir(targetPath), fileNameWithoutExt+".know.yml")

	knowledgeFile, err := s.knowledgeRepo.Read(knowYmlPath)
	if err == nil {
		for _, k := range knowledgeFile.KnowledgeList {
			k.Path = filepath.Clean(filepath.Join(filepath.Dir(targetPath), k.Path))
			knowledgeList = append(knowledgeList, k)
		}
	}

	return knowledgeList, nil
}
