package depsGraph

import (
	"encoding/json"
	"github.com/t-kuni/sisho/domain/repository/depsGraph"
	"os"
)

type repositoryImpl struct{}

func NewRepository() depsGraph.Repository {
	return &repositoryImpl{}
}

func (r *repositoryImpl) Read(path string) (depsGraph.DepsGraph, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var graph depsGraph.DepsGraph
	err = json.Unmarshal(content, &graph)
	if err != nil {
		return nil, err
	}

	return graph, nil
}

func (r *repositoryImpl) Write(path string, graph depsGraph.DepsGraph) error {
	content, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, content, 0644)
}
