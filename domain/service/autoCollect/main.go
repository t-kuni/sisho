package autoCollect

import (
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"path/filepath"
)

type AutoCollectService struct {
	configRepository   config.Repository
	contextScanService *contextScan.ContextScanService
}

func NewAutoCollectService(configRepository config.Repository, contextScanService *contextScan.ContextScanService) *AutoCollectService {
	return &AutoCollectService{
		configRepository:   configRepository,
		contextScanService: contextScanService,
	}
}

func (s *AutoCollectService) CollectAutoCollectFiles(rootDir string, targetPath string) ([]string, error) {
	cfg, err := s.configRepository.Read(filepath.Join(rootDir, "sisho.yml"))
	if err != nil {
		return nil, err
	}

	files, err := s.contextScanService.ContextScan(rootDir, targetPath)
	if err != nil {
		return nil, err
	}

	var collectedFiles []string
	for _, file := range files {
		baseName := filepath.Base(file)
		if cfg.AutoCollect.ReadmeMd && baseName == "README.md" {
			collectedFiles = append(collectedFiles, file)
		}
		if cfg.AutoCollect.TargetCodeMd && baseName == filepath.Base(targetPath)+".md" {
			collectedFiles = append(collectedFiles, file)
		}
	}

	return collectedFiles, nil
}
