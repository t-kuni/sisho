package depsGraphCommand

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/service/configFindService"
	"github.com/t-kuni/sisho/domain/service/projectScan"
	depsGraph2 "github.com/t-kuni/sisho/infrastructure/repository/depsGraph"
	file2 "github.com/t-kuni/sisho/infrastructure/repository/file"
	knowledge2 "github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"testing"
)

func TestDepsGraphCommand(t *testing.T) {
	callCommand := func(
		args []string,
	) error {
		fileRepo := file2.NewFileRepository()
		knowledgeRepo := knowledge2.NewRepository()
		depsGraphRepo := depsGraph2.NewRepository()
		configFindSvc := configFindService.NewConfigFindService(fileRepo)
		projectScanSvc := projectScan.NewProjectScanService(fileRepo)

		// コマンドの実行
		cmd := NewDepsGraphCommand(configFindSvc, projectScanSvc, knowledgeRepo, depsGraphRepo)
		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(cmd.CobraCommand)
		rootCmd.SetArgs(args)
		return rootCmd.Execute()
	}

	t.Run("依存グラフが正しく生成されること", func(t *testing.T) {
		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.MkDir(".sisho")

		// file1.go -> aaa/file2.go -> bbb/file3.go
		space.WriteFile("sisho.yml", []byte(""))
		space.WriteFile("file1.go", []byte(""))
		space.WriteFile("file1.go.know.yml", []byte(`
knowledge:
  - path: aaa/file2.go
    kind: implementations
    chain-make: true
`))
		space.WriteFile("aaa/file2.go", []byte(""))
		space.WriteFile("aaa/file2.go.know.yml", []byte(`
knowledge:
  - path: ../bbb/file3.go
    kind: implementations
    chain-make: true
`))
		space.WriteFile("bbb/file3.go", []byte(""))

		err := callCommand([]string{"deps-graph"})
		assert.NoError(t, err)

		// 生成された依存グラフを検証
		space.AssertFile(".sisho/deps-graph.json", func(actual []byte) {
			expect := `
{
  "aaa/file2.go": [ "file1.go" ],
  "bbb/file3.go": [ "aaa/file2.go" ]
}
`
			assert.JSONEq(t, expect, string(actual))
		})
	})

	t.Run("chain-makeがfalseのknowledgeは無視されること", func(t *testing.T) {
		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.MkDir(".sisho")

		space.WriteFile("sisho.yml", []byte(""))
		space.WriteFile("file1.go", []byte(""))
		space.WriteFile("file1.go.know.yml", []byte(`
knowledge:
  - path: aaa/file2.go
    kind: implementations
    chain-make: false
`))
		space.WriteFile("aaa/file2.go", []byte(""))

		err := callCommand([]string{"deps-graph"})
		assert.NoError(t, err)

		// 生成された依存グラフを検証
		space.AssertFile(".sisho/deps-graph.json", func(actual []byte) {
			expect := `{}`
			assert.JSONEq(t, expect, string(actual))
		})
	})

	t.Run(".sishoignoreに記載されているディレクトリ以下は処理されないこと", func(t *testing.T) {
		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.MkDir(".sisho")

		space.WriteFile("sisho.yml", []byte(""))
		space.WriteFile(".sishoignore", []byte("aaa\n"))

		space.WriteFile("aaa/file1.go", []byte(""))
		space.WriteFile("aaa/file1.go.know.yml", []byte(`
knowledge:
  - path: ../bbb/file2.go
    kind: implementations
    chain-make: true
`))
		space.WriteFile("bbb/file2.go", []byte(""))

		err := callCommand([]string{"deps-graph"})
		assert.NoError(t, err)

		// 生成された依存グラフを検証
		space.AssertFile(".sisho/deps-graph.json", func(actual []byte) {
			expect := `{}`
			assert.JSONEq(t, expect, string(actual))
		})
	})

	t.Run("隠しディレクトリ以下は処理されないこと", func(t *testing.T) {
		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.MkDir(".sisho")

		space.WriteFile("sisho.yml", []byte(""))

		space.WriteFile(".aaa/file1.go", []byte(""))
		space.WriteFile(".aaa/file1.go.know.yml", []byte(`
knowledge:
  - path: ../bbb/file2.go
    kind: implementations
    chain-make: true
`))
		space.WriteFile("bbb/file2.go", []byte(""))

		err := callCommand([]string{"deps-graph"})
		assert.NoError(t, err)

		// 生成された依存グラフを検証
		space.AssertFile(".sisho/deps-graph.json", func(actual []byte) {
			expect := `{}`
			assert.JSONEq(t, expect, string(actual))
		})
	})
}
