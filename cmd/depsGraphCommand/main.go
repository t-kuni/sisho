package depsGraphCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/repository/depsGraph"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/projectScan"
	"os"
	"path/filepath"
	"strings"
)

type DepsGraphCommand struct {
	CobraCommand *cobra.Command
}

func NewDepsGraphCommand(
	configFindService *configFindService.ConfigFindService,
	projectScanService *projectScan.ProjectScanService,
	knowledgeRepo knowledge.Repository,
	depsGraphRepo depsGraph.Repository,
) *DepsGraphCommand {
	cmd := &cobra.Command{
		Use:   "deps-graph",
		Short: "Generate dependency graph",
		Long:  `Generate dependency graph based on knowledge files`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find project root
			configPath, err := configFindService.FindConfig()
			if err != nil {
				return eris.Wrap(err, "failed to find config file")
			}

			// プロジェクトルートの絶対パス
			rootDir := configFindService.GetProjectRoot(configPath)

			// Scan project for knowledge files
			graph := make(depsGraph.DepsGraph)
			err = projectScanService.Scan(rootDir, func(pathFromRoot string, info os.FileInfo) error {
				if !info.IsDir() && strings.HasSuffix(pathFromRoot, ".know.yml") {
					yamlDirFromRoot := filepath.Dir(pathFromRoot)

					knowledgeFile, err := knowledgeRepo.Read(pathFromRoot)
					if err != nil {
						return eris.Wrapf(err, "failed to read knowledge file: %s", pathFromRoot)
					}

					for _, k := range knowledgeFile.KnowledgeList {
						if k.ChainMake {
							// 依存される側のパス（
							dependency := depsGraph.Dependency(filepath.Clean(filepath.Join(yamlDirFromRoot, k.Path)))
							// 依存する側のパス
							dependent := depsGraph.Dependent(strings.TrimSuffix(pathFromRoot, ".know.yml"))
							graph[dependency] = append(graph[dependency], dependent)
						}
					}
				}
				return nil
			})
			if err != nil {
				return eris.Wrap(err, "failed to scan project")
			}

			// Save dependency graph
			graphPath := filepath.Join(rootDir, ".sisho", "deps-graph.json")
			err = os.MkdirAll(filepath.Dir(graphPath), os.ModePerm)
			if err != nil {
				return eris.Wrapf(err, "failed to create directory: %s", filepath.Dir(graphPath))
			}
			err = depsGraphRepo.Write(graphPath, graph)
			if err != nil {
				return eris.Wrapf(err, "failed to write dependency graph to: %s", graphPath)
			}

			fmt.Printf("Dependency graph saved to %s\n", graphPath)
			return nil
		},
	}

	return &DepsGraphCommand{
		CobraCommand: cmd,
	}
}
