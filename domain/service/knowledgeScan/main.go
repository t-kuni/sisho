package knowledgeScan

import (
	"github.com/rotisserie/eris"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"os"
	"path/filepath"
)

type KnowledgeScanService struct {
	knowledgeRepo                 knowledge.Repository
	autoCollectService            *autoCollect.AutoCollectService
	knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService
}

func NewKnowledgeScanService(
	knowledgeRepo knowledge.Repository,
	autoCollectService *autoCollect.AutoCollectService,
	knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService,
) *KnowledgeScanService {
	return &KnowledgeScanService{
		knowledgeRepo:                 knowledgeRepo,
		autoCollectService:            autoCollectService,
		knowledgePathNormalizeService: knowledgePathNormalizeService,
	}
}

// ScanKnowledgeMultipleTarget performs a knowledge scan for multiple target paths
func (s *KnowledgeScanService) ScanKnowledgeMultipleTarget(rootDir string, targetPaths []string) ([]knowledge.Knowledge, error) {
	uniqueKnowledge := make(map[string]knowledge.Knowledge)

	for _, targetPath := range targetPaths {
		knowledgeList, err := s.ScanKnowledge(rootDir, targetPath)
		if err != nil {
			return nil, eris.Wrapf(err, "failed to scan knowledge for target: %s", targetPath)
		}

		// Remove duplicates
		for _, k := range knowledgeList {
			uniqueKnowledge[k.Path] = k
		}
	}

	var result []knowledge.Knowledge
	for _, k := range uniqueKnowledge {
		result = append(result, k)
	}

	return result, nil
}

// ScanKnowledge performs a knowledge scan for a single target path
func (s *KnowledgeScanService) ScanKnowledge(rootDir string, targetPath string) ([]knowledge.Knowledge, error) {
	var allKnowledge []knowledge.Knowledge

	// Scan layer knowledge list files (.knowledge.yml)
	knowledgeFromYml, err := s.scanKnowledgeYml(rootDir, targetPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to scan .knowledge.yml files")
	}
	allKnowledge = append(allKnowledge, knowledgeFromYml...)

	// Scan single file knowledge list files ([filename].know.yml)
	knowledgeFromKnowYml, err := s.scanKnowledgeKnowYml(rootDir, targetPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to scan .know.yml files")
	}
	allKnowledge = append(allKnowledge, knowledgeFromKnowYml...)

	// Process auto-collect
	autoCollectedFiles, err := s.autoCollectService.CollectAutoCollectFiles(rootDir, targetPath)
	if err != nil {
		return nil, eris.Wrap(err, "failed to auto-collect files")
	}
	for _, file := range autoCollectedFiles {
		allKnowledge = append(allKnowledge, knowledge.Knowledge{
			Path: file,
			Kind: "specifications", // Auto-collected files are treated as specifications
		})
	}

	// Normalize paths for all knowledge
	err = s.knowledgePathNormalizeService.NormalizePaths(rootDir, targetPath, &allKnowledge)
	if err != nil {
		return nil, eris.Wrap(err, "failed to normalize paths for all knowledge")
	}

	// Remove duplicates
	uniqueKnowledge := make(map[string]knowledge.Knowledge)
	for _, k := range allKnowledge {
		uniqueKnowledge[k.Path] = k
	}

	var result []knowledge.Knowledge
	for _, k := range uniqueKnowledge {
		result = append(result, k)
	}

	return result, nil
}

func (s *KnowledgeScanService) scanKnowledgeYml(rootDir string, targetPath string) ([]knowledge.Knowledge, error) {
	var knowledgeList []knowledge.Knowledge
	currentDir := filepath.Dir(targetPath)

	for {
		knowledgeFilePath := filepath.Join(currentDir, ".knowledge.yml")

		// Check if file exists before attempting to read
		_, err := os.Stat(knowledgeFilePath)
		if err == nil {
			knowledgeFile, err := s.knowledgeRepo.Read(knowledgeFilePath)
			if err != nil {
				return nil, eris.Wrap(err, "failed to read .knowledge.yml")
			}

			// Normalize paths
			err = s.knowledgePathNormalizeService.NormalizePaths(rootDir, knowledgeFilePath, &knowledgeFile.KnowledgeList)
			if err != nil {
				return nil, eris.Wrap(err, "failed to normalize paths for .knowledge.yml")
			}
			knowledgeList = append(knowledgeList, knowledgeFile.KnowledgeList...)
		} else if !os.IsNotExist(err) {
			return nil, eris.Wrap(err, "failed to check if .knowledge.yml exists")
		}

		// NOTE `currentDir == rootDir` という書き方は無限ループになるので禁止
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
	knowYmlPath := filepath.Join(filepath.Dir(targetPath), fileName+".know.yml")

	// Check if file exists before attempting to read
	_, err := os.Stat(knowYmlPath)
	if err == nil {
		knowledgeFile, err := s.knowledgeRepo.Read(knowYmlPath)
		if err != nil {
			return nil, eris.Wrap(err, "failed to read .know.yml")
		}
		knowledgeList = append(knowledgeList, knowledgeFile.KnowledgeList...)
	} else if !os.IsNotExist(err) {
		return nil, eris.Wrap(err, "failed to check if .know.yml exists")
	}

	return knowledgeList, nil
}
