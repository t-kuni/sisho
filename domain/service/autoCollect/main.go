package autoCollect

import (
	"github.com/t-kuni/sisho/domain/repository/config"
	"github.com/t-kuni/sisho/domain/service/contextScan"
	"os"
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

	var collectedFiles []string

	// README.mdの収集
	if cfg.AutoCollect.ReadmeMd {
		err = s.contextScanService.ContextScan(rootDir, targetPath, func(path string, info os.FileInfo) error {
			if filepath.Base(path) == "README.md" {
				collectedFiles = append(collectedFiles, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// [TARGET_CODE].mdの収集
	if cfg.AutoCollect.TargetCodeMd {
		targetMdPath := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".md")
		if _, err := os.Stat(targetMdPath); err == nil {
			collectedFiles = append(collectedFiles, targetMdPath)
		}
	}

	return collectedFiles, nil
}
