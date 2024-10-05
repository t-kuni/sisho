package knowledgePathNormalize

import (
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"path/filepath"
	"strings"
)

type KnowledgePathNormalizeService struct{}

func NewKnowledgePathNormalizeService() *KnowledgePathNormalizeService {
	return &KnowledgePathNormalizeService{}
}

func (s *KnowledgePathNormalizeService) NormalizePaths(projectRoot string, knowledgeFilePath string, knowledgeList *[]knowledge.Knowledge) error {
	knowledgeFileDir := filepath.Dir(knowledgeFilePath)

	for i, k := range *knowledgeList {
		absPath, err := s.NormalizePath(projectRoot, knowledgeFileDir, k.Path)
		if err != nil {
			return err
		}
		(*knowledgeList)[i].Path = absPath
	}

	return nil
}

func (s *KnowledgePathNormalizeService) NormalizePath(projectRoot, knowledgeFileDir, path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}

	if strings.HasPrefix(path, "@/") {
		return filepath.Join(projectRoot, strings.TrimPrefix(path, "@/")), nil
	}

	return filepath.Abs(filepath.Join(knowledgeFileDir, path))
}
