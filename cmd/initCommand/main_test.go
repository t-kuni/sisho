package initCommand

import (
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/infrastructure/repository/config"
	"github.com/t-kuni/sisho/testUtil"
	"go.uber.org/mock/gomock"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/repository/file"
)

func TestInitCommand(t *testing.T) {
	t.Run("sisho.ymlが作成されること", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		space := testUtil.BeginTestSpace(t)
		defer space.CleanUp()

		configRepo := config.NewConfigRepository()

		fileRepo := file.NewMockRepository(mockCtrl)
		fileRepo.EXPECT().Getwd().Return(space.Dir, nil).Times(1)

		initCmd := NewInitCommand(configRepo, fileRepo)

		cmd := &cobra.Command{}
		cmd.AddCommand(initCmd.CobraCommand)

		args := []string{"init"}
		cmd.SetArgs(args)

		err := initCmd.CobraCommand.Execute()
		assert.NoError(t, err)

		space.AssertFile("sisho.yml", func(actual []byte) {
			expect := `
lang: en
llm:
    driver: anthropic
    model: claude-3-5-sonnet-20240620
auto-collect:
    README.md: true
    "[TARGET_CODE].md": true
additional-knowledge:
    folder-structure: true
`
			assert.YAMLEq(t, expect, string(actual))
		})

		space.AssertExistPath(filepath.Join(".sisho", "history"))

		// Check if .gitignore was updated
		space.AssertFile(".gitignore", func(actual []byte) {
			expect := `/.sisho`
			assert.Contains(t, string(actual), expect)
		})
	})
}
