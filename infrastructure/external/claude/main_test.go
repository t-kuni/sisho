package claude

import (
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/t-kuni/sisho/domain/external/claude"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestClaudeClient_SendMessage_Integration(t *testing.T) {
	t.Skip()

	_, currentFile, _, _ := runtime.Caller(0)
	currentDir := filepath.Dir(currentFile)
	envPath := filepath.Join(currentDir, "..", "..", "..", ".env")
	err := godotenv.Load(envPath)
	assert.NoError(t, err)

	// APIキーが設定されていることを確認
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Fatal("ANTHROPIC_API_KEY is not set")
	}

	messages := []claude.Message{
		{Role: "user", Content: "こんにちは"},
	}
	model := "claude-3-5-sonnet-20240620"

	client := NewClaudeClient()
	result, err := client.SendMessage(messages, model)

	assert.NoError(t, err)
	assert.NotEmpty(t, result.Content)
	assert.NotEmpty(t, result.TerminationReason)
}
