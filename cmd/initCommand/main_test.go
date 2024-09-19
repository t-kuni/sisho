package initCommand

import (
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/infrastructure/repository/config"
	"go.uber.org/mock/gomock"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/t-kuni/sisho/domain/repository/file"
)

func TestInitCommand(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	tempDir, err := os.MkdirTemp("", "")
	defer os.RemoveAll(tempDir)
	assert.NoError(t, err)

	configRepo := config.NewConfigRepository()

	fileRepo := file.NewMockRepository(mockCtrl)
	fileRepo.EXPECT().Getwd().Return(tempDir, nil).Times(1)

	initCmd := NewInitCommand(configRepo, fileRepo)

	cmd := &cobra.Command{}
	cmd.AddCommand(initCmd.CobraCommand)

	args := []string{"init"}
	cmd.SetArgs(args)

	err = initCmd.CobraCommand.Execute()
	assert.NoError(t, err)

	// Check the content of sisho.yml
	configPath := filepath.Join(tempDir, "sisho.yml")
	config, err := configRepo.Read(configPath)
	assert.NoError(t, err)

	assert.Equal(t, "en", config.Lang)
	assert.Equal(t, "anthropic", config.LLM.Driver)
	assert.Equal(t, "claude-3-5-sonnet-20240620", config.LLM.Model)
	assert.True(t, config.AutoCollect.ReadmeMd)
	assert.True(t, config.AutoCollect.TargetCodeMd)
	assert.True(t, config.AdditionalKnowledge.FolderStructure)

	// Check if .sisho/history directory was created
	_, err = os.Stat(filepath.Join(tempDir, ".sisho", "history"))
	assert.NoError(t, err)

	// Check if .gitignore was updated
	gitignoreContent, err := os.ReadFile(filepath.Join(tempDir, ".gitignore"))
	assert.NoError(t, err)
	assert.Contains(t, string(gitignoreContent), "/.sisho")
}
