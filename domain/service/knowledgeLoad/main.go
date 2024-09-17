package knowledgeLoad

import (
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/prompts"
	"os"
	"path/filepath"
)

type KnowledgeLoadService struct {
	knowledgeRepo knowledge.Repository
}

func NewKnowledgeLoadService(knowledgeRepo knowledge.Repository) *KnowledgeLoadService {
	return &KnowledgeLoadService{
		knowledgeRepo: knowledgeRepo,
	}
}

func (s *KnowledgeLoadService) LoadKnowledge(rootDir string, knowledgeList []knowledge.Knowledge) ([]prompts.KnowledgeSet, error) {
	kindMap := make(map[string][]prompts.Knowledge)

	for _, k := range knowledgeList {
		content, err := s.readFile(filepath.Join(rootDir, k.Path))
		if err != nil {
			return nil, err
		}

		converted := prompts.Knowledge{
			Path:    k.Path,
			Content: content,
		}
		kindMap[string(k.Kind)] = append(kindMap[string(k.Kind)], converted)
	}

	var knowledgeSets []prompts.KnowledgeSet
	for kind, knowledges := range kindMap {
		knowledgeSets = append(knowledgeSets, prompts.KnowledgeSet{
			Kind:      kind,
			Knowledge: knowledges,
		})
	}

	return knowledgeSets, nil
}

func (s *KnowledgeLoadService) readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
