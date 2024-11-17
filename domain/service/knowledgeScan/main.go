package knowledgeScan

import (
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"
	"github.com/t-kuni/sisho/domain/model/kinds"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/autoCollect"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
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
			Kind: kinds.KindNameSpecifications, // Auto-collected files are treated as specifications
		})
	}

	// Normalize paths for all knowledge
	err = s.knowledgePathNormalizeService.NormalizePaths(rootDir, targetPath, &allKnowledge)
	if err != nil {
		return nil, eris.Wrap(err, "failed to normalize paths for all knowledge")
	}

	// Process knowledge-list kind
	allKnowledge, err = s.processKnowledgeListKind(rootDir, allKnowledge)
	if err != nil {
		return nil, eris.Wrap(err, "failed to process knowledge-list kind")
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

func (s *KnowledgeScanService) processKnowledgeListKind(rootDir string, knowledgeList []knowledge.Knowledge) ([]knowledge.Knowledge, error) {
	var result []knowledge.Knowledge

	for _, k := range knowledgeList {
		if k.Kind == kinds.KindNameKnowledgeList {
			additionalKnowledge, err := s.readKnowledgeListFile(rootDir, k.Path)
			if err != nil {
				return nil, eris.Wrapf(err, "failed to read knowledge-list file: %s", k.Path)
			}
			result = append(result, additionalKnowledge...)
		} else {
			result = append(result, k)
		}
	}

	return result, nil
}

func (s *KnowledgeScanService) readKnowledgeListFile(rootDir, path string) ([]knowledge.Knowledge, error) {
	knowledgeFile, err := s.knowledgeRepo.Read(path)
	if err != nil {
		return nil, eris.Wrap(err, "failed to read knowledge-list file")
	}

	err = s.knowledgePathNormalizeService.NormalizePaths(rootDir, path, &knowledgeFile.KnowledgeList)
	if err != nil {
		return nil, eris.Wrap(err, "failed to normalize paths for knowledge-list file")
	}

	var result []knowledge.Knowledge
	for _, k := range knowledgeFile.KnowledgeList {
		if k.Kind == kinds.KindNameKnowledgeList {
			additionalKnowledge, err := s.readKnowledgeListFile(rootDir, k.Path)
			if err != nil {
				return nil, eris.Wrapf(err, "failed to read nested knowledge-list file: %s", k.Path)
			}
			result = append(result, additionalKnowledge...)
		} else {
			result = append(result, k)
		}
	}

	return result, nil
}
