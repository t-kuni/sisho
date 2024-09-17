package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/t-kuni/sisho/domain/external/claude"
	"io"
	"os"
	"strings"
)

type ClaudeClient struct {
	apiKey string
}

func NewClaudeClient() *ClaudeClient {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		panic("Anthropic APIキーが設定されていません")
	}
	return &ClaudeClient{apiKey: apiKey}
}

func (c *ClaudeClient) SendMessage(messages []claude.Message, model string) (string, error) {
	client := resty.New()

	requestBody := ClaudeRequest{
		Model:     model,
		MaxTokens: 8192,
		Messages:  convertMessages(messages),
		Stream:    true,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := client.R().
		SetHeader("x-api-key", c.apiKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		SetDoNotParseResponse(true).
		Post("https://api.anthropic.com/v1/messages")

	if err != nil {
		return "", err
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode())
	}

	return processStreamResponse(resp.RawBody())
}

func convertMessages(messages []claude.Message) []Message {
	converted := make([]Message, len(messages))
	for i, msg := range messages {
		converted[i] = Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return converted
}

func processStreamResponse(body io.ReadCloser) (string, error) {
	reader := bufio.NewReader(body)
	var fullResponse strings.Builder

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}

		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))
		if len(data) == 0 {
			continue
		}

		var streamResp StreamResponse
		if err := json.Unmarshal(data, &streamResp); err != nil {
			return "", err
		}

		if streamResp.Type == "content_block_delta" {
			fullResponse.WriteString(streamResp.Delta.Text)
		}
	}

	return fullResponse.String(), nil
}

type ClaudeRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type StreamResponse struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
		Role string `json:"role"`
	} `json:"message"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
}
