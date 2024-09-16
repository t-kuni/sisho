package claude

// Client はClaude APIとの通信を抽象化するインターフェースです。
type Client interface {
	// SendMessage はメッセージを送信し、応答を返します。
	SendMessage(messages []Message) (string, error)
}

// Message はClaude APIに送信するメッセージの構造を表します。
type Message struct {
	Role    string
	Content string
}
