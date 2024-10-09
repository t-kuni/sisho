//go:generate mockgen -source=$GOFILE -destination=${GOFILE}_mock.go -package=$GOPACKAGE

package openAi

// Client はOpenAI APIとの通信を抽象化するインターフェースです。
type Client interface {
	// SendMessage はメッセージを送信し、応答を返します。
	// モデルのバリデーションは行いません。
	// ステータスコード200以外が返却された場合、レスポンスボディ全体をエラーメッセージに含めます。
	SendMessage(messages []Message, model string) (GenerationResult, error)
}

// Message はOpenAI APIに送信するメッセージの構造を表します。
type Message struct {
	Role    string
	Content string
}

// ModelName はOpenAI APIで使用可能なモデル名を定義する型です。
type ModelName string

const (
	ModelGPT4          ModelName = "gpt-4"
	ModelGPT4Turbo     ModelName = "gpt-4-turbo-preview"
	ModelGPT35Turbo    ModelName = "gpt-3.5-turbo"
	ModelGPT35Turbo16K ModelName = "gpt-3.5-turbo-16k"
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
		ModelGPT4,
		ModelGPT4Turbo,
		ModelGPT35Turbo,
		ModelGPT35Turbo16K,
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
