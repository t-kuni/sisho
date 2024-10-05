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

// ScanKnowledge performs a knowledge scan for the given target paths
func (s *KnowledgeScanService) ScanKnowledge(rootDir string, targetPaths []string) ([]knowledge.Knowledge, error) {
	uniqueKnowledge := make(map[string]knowledge.Knowledge)

	for _, targetPath := range targetPaths {
		// Scan layer knowledge list files (.knowledge.yml)
		knowledgeFromYml, err := s.scanKnowledgeYml(rootDir, targetPath)
		if err != nil {
			return nil, eris.Wrap(err, "failed to scan .knowledge.yml files")
		}

		// Scan single file knowledge list files ([filename].know.yml)
		knowledgeFromKnowYml, err := s.scanKnowledgeKnowYml(rootDir, targetPath)
		if err != nil {
			return nil, eris.Wrap(err, "failed to scan .know.yml files")
		}

		// Normalize paths for single file knowledge list files
		err = s.knowledgePathNormalizeService.NormalizePaths(rootDir, targetPath, &knowledgeFromKnowYml)
		if err != nil {
			return nil, eris.Wrap(err, "failed to normalize paths for .know.yml files")
		}

		// Process auto-collect
		autoCollectedFiles, err := s.autoCollectService.CollectAutoCollectFiles(rootDir, targetPath)
		if err != nil {
			return nil, eris.Wrap(err, "failed to auto-collect files")
		}

		// Combine all knowledge
		allKnowledge := append(knowledgeFromYml, knowledgeFromKnowYml...)
		for _, file := range autoCollectedFiles {
			allKnowledge = append(allKnowledge, knowledge.Knowledge{
				Path: file,
				Kind: "specifications", // Auto-collected files are treated as specifications
			})
		}

		// Remove duplicates
		for _, k := range allKnowledge {
			uniqueKnowledge[k.Path] = k
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
