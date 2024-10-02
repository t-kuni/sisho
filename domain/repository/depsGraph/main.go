package depsGraph

// Dependency 依存される側のパス（プロジェクトルートからの相対パス）
type Dependency string

// Dependent 依存する側のパス（プロジェクトルートからの相対パス）
type Dependent string

type DepsGraph map[Dependency][]Dependent

type Repository interface {
	Read(path string) (DepsGraph, error)
	Write(path string, graph DepsGraph) error
}
