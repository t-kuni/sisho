package local

import (
	_ "embed"
	"github.com/t-kuni/sisho/domain/model/chat"
)

//go:embed result.txt
var resultContent string

type LocalChat struct{}

func NewLocalChat() *LocalChat {
	return &LocalChat{}
}

func (l *LocalChat) Send(prompt string, model string) (chat.SendResult, error) {
	// Return the embedded content of result.txt as the response
	return chat.SendResult{
		Content:      resultContent,
		FinishReason: "stop",
	}, nil
}
