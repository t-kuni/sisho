package projectScan

import (
	"github.com/denormal/go-gitignore"
	"github.com/t-kuni/sisho/domain/repository/file"
	"os"
	"path/filepath"
)

type ProjectScanService struct {
	fileRepository file.Repository
}

func NewProjectScanService(fileRepository file.Repository) *ProjectScanService {
	return &ProjectScanService{
		fileRepository: fileRepository,
	}
}

type ScanFunc func(path string, info os.FileInfo) error

func (s *ProjectScanService) Scan(rootDir string, scanFunc ScanFunc) error {
	// Load .sishoignore file
	ignorePath := filepath.Join(rootDir, ".sishoignore")
	ignore, err := gitignore.NewFromFile(ignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && info.Name()[0] == '.' {
			return filepath.SkipDir
		}

		// Check if the path should be ignored
		if ignore != nil {
			relPath, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			if ignore.Match(relPath) != nil {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Call the scan function for each file/directory
		return scanFunc(path, info)
	})
}
