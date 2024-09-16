package openAi

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"os"
	"strings"
)

type OpenAiChat struct {
	history []Message
}

func (c *OpenAiChat) addToHistory(m Message) {
	c.history = append(c.history, m)
}

type OpenAiRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAiResponseStreamChoice struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (c *OpenAiChat) Send(prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI APIキーが設定されていません")
	}

	client := resty.New()

	// 新しいユーザーのメッセージを履歴に追加
	c.addToHistory(Message{
		Role:    "user",
		Content: prompt,
	})

	requestBody := OpenAiRequest{
		Model:    "chatgpt-4o-latest",
		Messages: c.history, // 履歴全体をリクエストに含める
		Stream:   true,      // ストリーミングを有効にする
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// ストリームを処理する
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		SetDoNotParseResponse(true). // レスポンスを手動でパース
		Post("https://api.openai.com/v1/chat/completions")
	if err != nil {
		return "", err
	}
	defer resp.RawBody().Close()

	reader := bufio.NewReader(resp.RawBody())
	var fullResponse strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		// データ行のみを処理する
		if strings.HasPrefix(line, "data: ") {
			line = strings.TrimPrefix(line, "data: ")
			if strings.TrimSpace(line) == "[DONE]" {
				break
			}

			var streamResponse OpenAiResponseStreamChoice
			if err := json.Unmarshal([]byte(line), &streamResponse); err != nil {
				return "", err
			}

			// ストリームデータのコンテンツを追加
			if len(streamResponse.Choices) > 0 {
				fullResponse.WriteString(streamResponse.Choices[0].Delta.Content)
				fmt.Print(streamResponse.Choices[0].Delta.Content) // 逐次出力
			}
		}
	}

	// AIからの返答を履歴に追加
	c.addToHistory(Message{
		Role:    "assistant",
		Content: fullResponse.String(),
	})

	return fullResponse.String(), nil
}
