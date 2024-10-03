package folderStructureMake

import (
	"github.com/denormal/go-gitignore"
	"github.com/rotisserie/eris"
	"os"
	"path/filepath"
	"strings"
)

type FolderStructureMakeService struct {
}

func NewFolderStructureMakeService() *FolderStructureMakeService {
	return &FolderStructureMakeService{}
}

func (s *FolderStructureMakeService) MakeTree(rootPath string) (string, error) {
	ignoreFile := filepath.Join(rootPath, ".sishoignore")
	var ignore gitignore.GitIgnore
	var err error

	if _, err := os.Stat(ignoreFile); err == nil {
		ignore, err = gitignore.NewFromFile(ignoreFile)
		if err != nil {
			return "", eris.Wrap(err, "failed to read .sishoignore file")
		}
	}

	var result strings.Builder
	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return eris.Wrap(err, "error walking through directory")
		}

		relativePath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return eris.Wrap(err, "failed to get relative path")
		}

		if relativePath != "." && ignore != nil && ignore.Match(relativePath) != nil {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		indent := strings.Repeat("  ", strings.Count(relativePath, string(os.PathSeparator)))
		if info.IsDir() {
			result.WriteString(indent + "/" + info.Name() + "\n")
		} else {
			result.WriteString(indent + info.Name() + "\n")
		}

		return nil
	})

	if err != nil {
		return "", eris.Wrap(err, "failed to walk directory")
	}

	return result.String(), nil
}
