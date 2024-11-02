//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package configFindService

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/t-kuni/sisho/util/path"
)

type ConfigFindService struct {
	fileRepository FileRepository
}

type FileRepository interface {
	Getwd() (string, error)
}

func NewConfigFindService(fileRepository FileRepository) *ConfigFindService {
	return &ConfigFindService{
		fileRepository: fileRepository,
	}
}

func (s *ConfigFindService) FindConfig() (string, error) {
	currentDir, err := s.fileRepository.Getwd()
	if err != nil {
		return "", err
	}

	currentDir, err = path.AfterGetAbsPath(currentDir)
	if err != nil {
		return "", err
	}

	for {
		ymlPath := filepath.Join(currentDir, "sisho.yml")
		yamlPath := filepath.Join(currentDir, "sisho.yaml")

		if exists(ymlPath) {
			return ymlPath, nil
		}
		if exists(yamlPath) {
			return yamlPath, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			break
		}
		currentDir = parentDir
	}

	return "", errors.New("sisho.yml または sisho.yaml が見つかりませんでした")
}

func (s *ConfigFindService) GetProjectRoot(configPath string) string {
	return filepath.Dir(configPath)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
