package knowledgePathNormalize

import (
	"path/filepath"
	"strings"

	"github.com/t-kuni/sisho/domain/repository/knowledge"
	pathUtil "github.com/t-kuni/sisho/util/path"
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
	var err error
	if filepath.IsAbs(path) {
		path, err = pathUtil.AfterGetAbsPath(path)
		if err != nil {
			return path, err
		}
		return path, nil
	}

	if strings.HasPrefix(path, "@/") {
		return filepath.Join(projectRoot, strings.TrimPrefix(path, "@/")), nil
	}

	path, err = filepath.Abs(filepath.Join(knowledgeFileDir, path))
	if err != nil {
		return path, err
	}

	return path, nil
}
