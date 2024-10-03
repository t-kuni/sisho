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
		Long:  `Scan the project and generate a dependency graph based on knowledge files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepsGraph(configFindService, projectScanService, knowledgeRepo, depsGraphRepo)
		},
	}

	return &DepsGraphCommand{
		CobraCommand: cmd,
	}
}

func runDepsGraph(
	configFindService *configFindService.ConfigFindService,
	projectScanService *projectScan.ProjectScanService,
	knowledgeRepo knowledge.Repository,
	depsGraphRepo depsGraph.Repository,
) error {
	// Find config and get project root
	configPath, err := configFindService.FindConfig()
	if err != nil {
		return err
	}
	rootDir := configFindService.GetProjectRoot(configPath)

	// Initialize dependency graph
	graph := make(depsGraph.DepsGraph)

	// Scan project
	err = projectScanService.Scan(rootDir, func(pathFromRoot string, info os.FileInfo) error {
		// ここは必ずstrings.HasSuffixを使う必要がある
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
	}, func(event string, path string) {
		if event == "scan_file" {
			return
		}
		fmt.Printf("%s: %s\n", event, path)
	})

	if err != nil {
		return err
	}

	// Save dependency graph
	outputPath := filepath.Join(rootDir, ".sisho", "deps-graph.json")
	err = os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		return err
	}

	err = depsGraphRepo.Write(outputPath, graph)
	if err != nil {
		return err
	}

	fmt.Printf("Dependency graph has been saved to %s\n", outputPath)
	return nil
}
