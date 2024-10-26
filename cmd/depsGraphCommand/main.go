package depsGraphCommand

import (
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/repository/depsGraph"
	"github.com/t-kuni/sisho/domain/repository/knowledge"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/knowledgePathNormalize"
	"github.com/t-kuni/sisho/domain/service/projectScan"
	"github.com/t-kuni/sisho/util/path"
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
	knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService,
) *DepsGraphCommand {
	cmd := &cobra.Command{
		Use:   "deps-graph",
		Short: "Generate dependency graph",
		Long:  `Scan the project and generate a dependency graph based on knowledge files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDepsGraph(configFindService, projectScanService, knowledgeRepo, depsGraphRepo, knowledgePathNormalizeService)
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
	knowledgePathNormalizeService *knowledgePathNormalize.KnowledgePathNormalizeService,
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
			knowledgeFile, err := knowledgeRepo.Read(filepath.Join(rootDir, pathFromRoot))
			if err != nil {
				return eris.Wrapf(err, "failed to read knowledge file: %s", pathFromRoot)
			}

			for i := range knowledgeFile.KnowledgeList {
				normalizedPath, err := knowledgePathNormalizeService.NormalizePath(rootDir, filepath.Dir(pathFromRoot), knowledgeFile.KnowledgeList[i].Path)
				if err != nil {
					return eris.Wrapf(err, "failed to normalize path: %s", knowledgeFile.KnowledgeList[i].Path)
				}
				knowledgeFile.KnowledgeList[i].Path = normalizedPath
			}

			for _, k := range knowledgeFile.KnowledgeList {
				if k.ChainMake {
					// 依存される側のパス
					relPath, err := filepath.Rel(rootDir, k.Path)
					if err != nil {
						return eris.Wrapf(err, "failed to get relative path: %s", k.Path)
					}
					dependency := depsGraph.Dependency(path.BeforeWrite(filepath.Clean(relPath)))
					// 依存する側のパス
					dependent := depsGraph.Dependent(path.BeforeWrite(strings.TrimSuffix(pathFromRoot, ".know.yml")))
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
