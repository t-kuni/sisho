package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Lang string `yaml:"lang"`
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
	// カレントディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// ルートディレクトリまでループしてファイルを探す
	for {
		// `llm-config.yml` または `llm-config.yaml` が存在するかをチェック
		ymlPath := filepath.Join(currentDir, "llm-config.yml")
		yamlPath := filepath.Join(currentDir, "llm-config.yaml")

		if _, err := os.Stat(ymlPath); err == nil {
			return ymlPath, nil
		}
		if _, err := os.Stat(yamlPath); err == nil {
			return yamlPath, nil
		}

		// ルートディレクトリに達したらエラーを返す
		if currentDir == filepath.Dir(currentDir) {
			break
		}

		// 親ディレクトリに移動
		currentDir = filepath.Dir(currentDir)
	}

	return "", errors.New("llm-config.ymlまたはllm-config.yamlが見つかりませんでした")
}
