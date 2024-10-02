package autoCollect

import (
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"os"
	"path/filepath"
	"strings"
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

	var collectedFiles []string
	targetName := strings.TrimSuffix(filepath.Base(targetPath), filepath.Ext(targetPath))

	err = s.contextScanService.ContextScan(rootDir, targetPath, func(path string, info os.FileInfo) error {
		baseName := filepath.Base(path)
		if cfg.AutoCollect.ReadmeMd && baseName == "README.md" {
			collectedFiles = append(collectedFiles, path)
		}
		if cfg.AutoCollect.TargetCodeMd && baseName == targetName+".md" {
			collectedFiles = append(collectedFiles, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return collectedFiles, nil
}
