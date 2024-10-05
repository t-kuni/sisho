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

// CollectAutoCollectFiles collects files based on the auto-collect settings in sisho.yml
// It returns a slice of absolute file paths
func (s *AutoCollectService) CollectAutoCollectFiles(rootDir string, targetPath string) ([]string, error) {
	cfg, err := s.configRepository.Read(filepath.Join(rootDir, "sisho.yml"))
	if err != nil {
		return nil, err
	}

	var collectedFiles []string

	// Collect README.md files
	if cfg.AutoCollect.ReadmeMd {
		err = s.contextScanService.ContextScan(rootDir, targetPath, func(path string, info os.FileInfo) error {
			if filepath.Base(path) == "README.md" {
				absPath, err := filepath.Abs(filepath.Join(rootDir, path))
				if err != nil {
					return err
				}
				collectedFiles = append(collectedFiles, absPath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Collect [TARGET_CODE].md file
	if cfg.AutoCollect.TargetCodeMd {
		targetMdPath := filepath.Join(filepath.Dir(targetPath), filepath.Base(targetPath)+".md")
		absTargetMdPath, err := filepath.Abs(targetMdPath)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absTargetMdPath); err == nil {
			collectedFiles = append(collectedFiles, absTargetMdPath)
		}
	}

	return collectedFiles, nil
}
