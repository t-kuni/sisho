package openAi

import (
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"os"
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
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAiResponse struct {
	Choices []OpenAiResponseChoice `json:"choices"`
}

type OpenAiResponseChoice struct {
	Message OpenAiResponseChoiceMessage `json:"message"`
}

type OpenAiResponseChoiceMessage struct {
	Content string `json:"content"`
}

func (c *OpenAiChat) Send(prompt string, model string) (string, error) {
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
		Model:    model,
		Messages: c.history, // 履歴全体をリクエストに含める
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	var aiResponse OpenAiResponse

	_, err = client.R().
		SetHeader("Authorization", "Bearer "+apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		SetResult(&aiResponse). // レスポンスを構造体にマッピング
		Post("https://api.openai.com/v1/chat/completions")
	if err != nil {
		return "", err
	}

	// AIからの返答を履歴に追加
	c.addToHistory(Message{
		Role:    "assistant",
		Content: aiResponse.Choices[0].Message.Content,
	})

	return aiResponse.Choices[0].Message.Content, nil
}
