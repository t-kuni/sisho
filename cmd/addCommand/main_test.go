package addCommand

import (
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/infrastructure/repository/knowledge"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"testing"

	"github.com/spf13/cobra"
)

func TestAddCommand(t *testing.T) {
	t.Run(".knowledge.ymlが作成されること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.WriteFile("aaa/bbb/test.txt", []byte("test content"))

		mockKnowledgeRepo := knowledge.NewRepository()

		addCmd := NewAddCommand(mockKnowledgeRepo)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(addCmd.CobraCommand)

		rootCmd.SetArgs([]string{"add", "examples", "aaa/bbb/test.txt"})
		err := rootCmd.Execute()
		assert.NoError(t, err)

		space.AssertFile(".knowledge.yml", func(actual []byte) {
			expect := `
knowledge:
  - path: aaa/bbb/test.txt
    kind: examples
`
			assert.YAMLEq(t, expect, string(actual))
		})
	})

	t.Run("既存の.knowledge.ymlにpathが追加されること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		space.WriteFile("aaa/bbb/test.txt", []byte("test content"))
		space.WriteFile(".knowledge.yml", []byte(`
knowledge:
  - path: ccc/ddd/another.txt
    kind: implementations
`))

		mockKnowledgeRepo := knowledge.NewRepository()

		addCmd := NewAddCommand(mockKnowledgeRepo)

		rootCmd := &cobra.Command{}
		rootCmd.AddCommand(addCmd.CobraCommand)

		rootCmd.SetArgs([]string{"add", "examples", "aaa/bbb/test.txt"})
		err := rootCmd.Execute()
		assert.NoError(t, err)

		space.AssertFile(".knowledge.yml", func(actual []byte) {
			expect := `
knowledge:
  - path: ccc/ddd/another.txt
    kind: implementations
  - path: aaa/bbb/test.txt
    kind: examples
`
			assert.YAMLEq(t, expect, string(actual))
		})
	})
}
