//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package claude

// Client はClaude APIとの通信を抽象化するインターフェースです。
type Client interface {
	// SendMessage はメッセージを送信し、応答を返します。
	// モデルのバリデーションは行いません。
	// ステータスコード200以外が返却された場合、レスポンスボディ全体をエラーメッセージに含めます。
	SendMessage(messages []Message, model string) (GenerationResult, error)
}

// Message はClaude APIに送信するメッセージの構造を表します。
type Message struct {
	Role    string
	Content string
}

// ModelName はClaude APIで使用可能なモデル名を定義する型です。
type ModelName string

const (
	ModelClaude3Opus   ModelName = "claude-3-opus-20240229"
	ModelClaude3Sonnet ModelName = "claude-3-sonnet-20240229"
	ModelClaude3Haiku  ModelName = "claude-3-haiku-20240307"
)

// NewMessage は新しいMessageインスタンスを作成します。
func NewMessage(role, content string) Message {
	return Message{
		Role:    role,
		Content: content,
	}
}

// GetAvailableModels は利用可能なすべてのモデル名を返します。
func GetAvailableModels() []ModelName {
	return []ModelName{
		ModelClaude3Opus,
		ModelClaude3Sonnet,
		ModelClaude3Haiku,
	}
}

// ValidateModel は指定されたモデル名が有効かどうかを検証します。
func ValidateModel(model string) bool {
	for _, validModel := range GetAvailableModels() {
		if string(validModel) == model {
			return true
		}
	}
	return false
}

// GenerationResult は生成結果を表す構造体です。
type GenerationResult struct {
	Content           string
	TerminationReason string
}
