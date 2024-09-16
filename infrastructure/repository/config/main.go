package config

import (
	"github.com/t-kuni/sisho/domain/repository/config"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type ConfigRepository struct{}

func NewConfigRepository() *ConfigRepository {
	return &ConfigRepository{}
}

func (r *ConfigRepository) Read(path string) (*config.Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg config.Config
	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (r *ConfigRepository) Write(path string, cfg *config.Config) error {
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}
