package contextScan

import (
	"github.com/t-kuni/sisho/domain/repository/file"
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
func (s *ContextScanService) ContextScan(rootDir string, targetPath string) ([]string, error) {
	var collectedFiles []string
	currentDir, err := filepath.Abs(filepath.Dir(targetPath))
	if err != nil {
		return nil, err
	}
	rootDir, err = filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	for {
		// Collect README.md
		readmePath := filepath.Join(currentDir, "README.md")
		if s.fileRepository.Exists(readmePath) {
			relPath, err := filepath.Rel(rootDir, readmePath)
			if err != nil {
				return nil, err
			}
			collectedFiles = append(collectedFiles, relPath)
		}

		// Collect [TARGET_CODE].md
		targetCodeMdPath := filepath.Join(currentDir, filepath.Base(targetPath)+".md")
		if s.fileRepository.Exists(targetCodeMdPath) {
			relPath, err := filepath.Rel(rootDir, targetCodeMdPath)
			if err != nil {
				return nil, err
			}
			collectedFiles = append(collectedFiles, relPath)
		}

		if currentDir == rootDir {
			break
		}
		currentDir = filepath.Dir(currentDir)
	}

	return collectedFiles, nil
}
