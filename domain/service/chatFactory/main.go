package chatFactory

import (
	"github.com/rotisserie/eris"
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/model/chat"
	modelClaude "github.com/t-kuni/sisho/domain/model/chat/claude"
	"github.com/t-kuni/sisho/domain/model/chat/local"
	modelOpenAi "github.com/t-kuni/sisho/domain/model/chat/openAi"
	"github.com/t-kuni/sisho/domain/repository/config"
)

type ChatFactory struct {
	openAiClient openAi.Client
	claudeClient claude.Client
}

func NewChatFactory(openAiClient openAi.Client, claudeClient claude.Client) *ChatFactory {
	return &ChatFactory{
		openAiClient: openAiClient,
		claudeClient: claudeClient,
	}
}

func (s *ChatFactory) Make(cfg *config.Config) (chat.Chat, error) {
	var c chat.Chat
	var err error

	switch cfg.LLM.Driver {
	case "open-ai":
		c = modelOpenAi.NewOpenAiChat(s.openAiClient)
	case "anthropic":
		c = modelClaude.NewClaudeChat(s.claudeClient)
	case "local":
		c = local.NewLocalChat()
	default:
		return nil, eris.Errorf("unsupported LLM driver: %s", cfg.LLM.Driver)
	}

	return c, err
}
