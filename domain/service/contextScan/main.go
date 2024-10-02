package contextScan

import (
	"github.com/t-kuni/sisho/domain/repository/file"
	"os"
	"path/filepath"
)

type ContextScanService struct {
	fileRepository file.Repository
}

func NewContextScanService(fileRepository file.Repository) *ContextScanService {
	return &ContextScanService{
		fileRepository: fileRepository,
	}
}

// ContextScan scans the directory structure from the target path up to the root directory
// and collects relevant files (README.md and [TARGET_CODE].md).
func (s *ContextScanService) ContextScan(rootDir string, targetPath string, scanFunc func(path string, info os.FileInfo) error) error {
	currentDir, err := filepath.Abs(filepath.Dir(targetPath))
	if err != nil {
		return err
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return err
	}

	for {
		files, err := os.ReadDir(currentDir)
		if err != nil {
			return err
		}

		for _, file := range files {
			filePath := filepath.Join(currentDir, file.Name())
			relPath, err := filepath.Rel(rootDir, filePath)
			if err != nil {
				return err
			}

			info, err := file.Info()
			if err != nil {
				return err
			}

			if err := scanFunc(relPath, info); err != nil {
				return err
			}
		}

		if currentDir == rootDir {
			break
		}
		currentDir = filepath.Dir(currentDir)
	}

	return nil
}
