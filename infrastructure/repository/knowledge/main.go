package knowledge

import (
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"gopkg.in/yaml.v3"
	"os"
)

type repositoryImpl struct{}

func NewRepository() knowledge.Repository {
	return &repositoryImpl{}
}

func (r *repositoryImpl) Read(path string) (knowledge.KnowledgeFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return knowledge.KnowledgeFile{}, err
	}

	var knowledgeFile knowledge.KnowledgeFile
	err = yaml.Unmarshal(content, &knowledgeFile)
	if err != nil {
		return knowledge.KnowledgeFile{}, err
	}

	return knowledgeFile, nil
}

func (r *repositoryImpl) Write(path string, knowledgeFile knowledge.KnowledgeFile) error {
	content, err := yaml.Marshal(knowledgeFile)
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}
