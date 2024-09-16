package configFindService

import (
	"errors"
	"os"
	"path/filepath"
)

type ConfigFindService struct {
	fileRepository FileRepository
}

type FileRepository interface {
	Exists(path string) bool
}

func NewConfigFindService(fileRepository FileRepository) *ConfigFindService {
	return &ConfigFindService{
		fileRepository: fileRepository,
	}
}

func (s *ConfigFindService) FindConfig() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		ymlPath := filepath.Join(currentDir, "sisho.yml")
		yamlPath := filepath.Join(currentDir, "sisho.yaml")

		if s.fileRepository.Exists(ymlPath) {
			return ymlPath, nil
		}
		if s.fileRepository.Exists(yamlPath) {
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
