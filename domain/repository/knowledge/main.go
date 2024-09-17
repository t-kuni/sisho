package knowledge

import (
	"github.com/t-kuni/sisho/domain/model/kinds"
)

type Knowledge struct {
	Path string
	Kind kinds.KindName
}

type KnowledgeFile struct {
	KnowledgeList []Knowledge `yaml:"knowledge"`
}

type Repository interface {
	Read(path string) (KnowledgeFile, error)
	Write(path string, knowledgeFile KnowledgeFile) error
}
