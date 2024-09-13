package chat

type Chat interface {
	Send(prompt string) (string, error)
}
