//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package chat

type Chat interface {
	Send(prompt string, model string) (SendResult, error)
}

type Message struct {
	Role    string
	Content string
}

type ChatWithHistory interface {
	Chat
	GetHistory() []Message
}

// SendResult represents the result of a chat interaction
type SendResult struct {
	Content      string
	FinishReason string
}
