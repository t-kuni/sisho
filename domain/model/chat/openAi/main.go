package openai

import (
	"github.com/t-kuni/sisho/domain/external/openAi"
	"github.com/t-kuni/sisho/domain/model/chat"
)

type OpenAiChat struct {
	client  openAi.Client
	history []chat.Message
}

func NewOpenAiChat(client openAi.Client) *OpenAiChat {
	return &OpenAiChat{
		client:  client,
		history: []chat.Message{},
	}
}

func (o *OpenAiChat) Send(prompt string, model string) (string, error) {
	// Add user message to history
	o.history = append(o.history, chat.Message{Role: "user", Content: prompt})

	// Convert history to OpenAI messages
	openAiMessages := make([]openAi.Message, len(o.history))
	for i, msg := range o.history {
		openAiMessages[i] = openAi.Message{Role: msg.Role, Content: msg.Content}
	}

	// Send message to OpenAI API
	response, err := o.client.SendMessage(openAiMessages, model)
	if err != nil {
		return "", err
	}

	// Add assistant response to history
	o.history = append(o.history, chat.Message{Role: "assistant", Content: response})

	return response, nil
}

func (o *OpenAiChat) GetHistory() []chat.Message {
	return o.history
}
