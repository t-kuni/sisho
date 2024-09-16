package claude

import (
	externalClaude "github.com/t-kuni/sisho/domain/external/claude"
)

type ClaudeChat struct {
	history []Message
	client  externalClaude.Client
}

func NewClaudeChat(client externalClaude.Client) *ClaudeChat {
	return &ClaudeChat{
		history: []Message{},
		client:  client,
	}
}

func (c *ClaudeChat) addToHistory(m Message) {
	c.history = append(c.history, m)
}

type Message struct {
	Role    string
	Content string
}

func (c *ClaudeChat) Send(prompt string) (string, error) {
	c.addToHistory(Message{
		Role:    "user",
		Content: prompt,
	})

	messages := make([]externalClaude.Message, len(c.history))
	for i, msg := range c.history {
		messages[i] = externalClaude.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	responseText, err := c.client.SendMessage(messages)
	if err != nil {
		return "", err
	}

	c.addToHistory(Message{
		Role:    "assistant",
		Content: responseText,
	})

	return responseText, nil
}
