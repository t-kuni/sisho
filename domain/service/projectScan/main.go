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

// ScanFunc
// path はrootDirからの相対パス
type ScanFunc func(path string, info os.FileInfo) error

// ProgressFunc は進捗を通知するための関数型です
type ProgressFunc func(event string, path string)

func (s *ProjectScanService) Scan(rootDir string, scanFunc ScanFunc, progressFunc ProgressFunc) error {
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

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && info.Name()[0] == '.' {
			progressFunc("skip_dir", relPath)
			return filepath.SkipDir
		}

		// Check if the path should be ignored
		if ignore != nil {
			if relPath != "." && ignore.Match(relPath) != nil {
				if info.IsDir() {
					progressFunc("skip_ignored_dir", relPath)
					return filepath.SkipDir
				}
				progressFunc("skip_ignored_file", relPath)
				return nil
			}
		}

		if info.IsDir() {
			progressFunc("enter_dir", relPath)
		} else {
			progressFunc("scan_file", relPath)
		}

		// Call the scan function with relative path
		return scanFunc(relPath, info)
	})
}
