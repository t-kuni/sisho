package claude

import (
	"github.com/t-kuni/sisho/domain/external/claude"
	"github.com/t-kuni/sisho/domain/model/chat"
)

type ClaudeChat struct {
	client  claude.Client
	history []chat.Message
}

func NewClaudeChat(client claude.Client) *ClaudeChat {
	return &ClaudeChat{
		client:  client,
		history: []chat.Message{},
	}
}

func (c *ClaudeChat) Send(prompt string, model string) (chat.SendResult, error) {
	// Add user message to history
	c.history = append(c.history, chat.Message{Role: "user", Content: prompt})

	// Convert history to Claude messages
	claudeMessages := make([]claude.Message, len(c.history))
	for i, msg := range c.history {
		claudeMessages[i] = claude.Message{Role: msg.Role, Content: msg.Content}
	}

	// Send message to Claude API
	response, err := c.client.SendMessage(claudeMessages, model)
	if err != nil {
		return chat.SendResult{}, err
	}

	// Add assistant response to history
	c.history = append(c.history, chat.Message{Role: "assistant", Content: response.Content})

	return chat.SendResult{
		Content:      response.Content,
		FinishReason: response.TerminationReason,
	}, nil
}

func (c *ClaudeChat) GetHistory() []chat.Message {
	return c.history
}
