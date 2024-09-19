//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package config

type Config struct {
	Lang                string              `yaml:"lang"`
	LLM                 LLM                 `yaml:"llm"`
	AutoCollect         AutoCollect         `yaml:"auto-collect"`
	AdditionalKnowledge AdditionalKnowledge `yaml:"additional-knowledge"`
}

type LLM struct {
	Driver string `yaml:"driver"`
	Model  string `yaml:"model"`
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
