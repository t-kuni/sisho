package chat

type Chat interface {
	Send(prompt string, model string) (string, error)
}

type Message struct {
	Role    string
	Content string
}

type ChatWithHistory interface {
	Chat
	GetHistory() []Message
}
