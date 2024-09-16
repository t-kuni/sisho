package config

type Config struct {
	Lang                string              `yaml:"lang"`
	AutoCollect         AutoCollect         `yaml:"auto-collect"`
	AdditionalKnowledge AdditionalKnowledge `yaml:"additional-knowledge"`
}

type AutoCollect struct {
	ReadmeMd     bool `yaml:"README.md"`
	TargetCodeMd bool `yaml:"[TARGET_CODE].md"`
}

type AdditionalKnowledge struct {
	FolderStructure bool `yaml:"folder-structure"`
}

type Repository interface {
	Read(path string) (*Config, error)
	Write(path string, cfg *Config) error
}
