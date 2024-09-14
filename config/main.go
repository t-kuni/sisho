package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Lang        string      `yaml:"lang"`
	AutoCollect AutoCollect `yaml:"auto-collect"`
}

type AutoCollect struct {
	ReadmeMd     bool `yaml:"README.md"`
	TargetCodeMd bool `yaml:"[TARGET_CODE].md"`
}

type ConfigHolder struct {
	Path    string
	RootDir string
	Config
}

func ReadConfig() (ConfigHolder, error) {
	path, err := findConfig()
	if err != nil {
		return ConfigHolder{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return ConfigHolder{}, err
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return ConfigHolder{}, err
	}

	return ConfigHolder{
		Path:    path,
		Config:  config,
		RootDir: filepath.Dir(path),
	}, nil
}

func WriteConfig(holder ConfigHolder) error {
	content, err := yaml.Marshal(holder.Config)
	if err != nil {
		return err
	}

	err = os.WriteFile(holder.Path, content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func findConfig() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		ymlPath := filepath.Join(currentDir, "sisho.yml")
		yamlPath := filepath.Join(currentDir, "sisho.yaml")

		if _, err := os.Stat(ymlPath); err == nil {
			return ymlPath, nil
		}
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, nil
		}

		if currentDir == filepath.Dir(currentDir) {
			break
		}

		currentDir = filepath.Dir(currentDir)
	}

	return "", errors.New("sisho.ymlまたはsisho.yamlが見つかりませんでした")
}

func ContextScan(rootDir string, targetPath string) ([]string, error) {
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
		// README.mdの収集
		readmePath := filepath.Join(currentDir, "README.md")
		if _, err := os.Stat(readmePath); err == nil {
			collectedFiles = append(collectedFiles, readmePath)
		}

		// [TARGET_CODE].mdの収集
		targetCodeMdPath := filepath.Join(currentDir, filepath.Base(targetPath)+".md")
		if _, err := os.Stat(targetCodeMdPath); err == nil {
			collectedFiles = append(collectedFiles, targetCodeMdPath)
		}

		if currentDir == rootDir {
			break
		}
		currentDir = filepath.Dir(currentDir)
	}

	return collectedFiles, nil
}

// CollectAutoCollectFiles関数を修正
func CollectAutoCollectFiles(config Config, rootDir string, targetPath string) ([]string, error) {
	files, err := ContextScan(rootDir, targetPath)
	if err != nil {
		return nil, err
	}

	var collectedFiles []string
	for _, file := range files {
		baseName := filepath.Base(file)
		if config.AutoCollect.ReadmeMd && baseName == "README.md" {
			collectedFiles = append(collectedFiles, file)
		}
		if config.AutoCollect.TargetCodeMd && baseName == filepath.Base(targetPath)+".md" {
			collectedFiles = append(collectedFiles, file)
		}
	}

	return collectedFiles, nil
}
