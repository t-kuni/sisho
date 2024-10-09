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

// NewClaudeClient initializes a new client for Claude API with necessary settings.
func NewClaudeClient() *ClaudeClient {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		panic("Anthropic APIキーが設定されていません")
	}
	return &ClaudeClient{apiKey: apiKey}
}

// SendMessage sends an array of Message to Claude API and waits for a response.
func (c *ClaudeClient) SendMessage(messages []claude.Message, model string) (claude.GenerationResult, error) {
	client := resty.New()

	requestBody := ClaudeRequest{
		Model:     model,
		MaxTokens: 8192,
		Messages:  convertMessages(messages),
		Stream:    true,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return claude.GenerationResult{}, err
	}

	resp, err := client.R().
		SetHeader("x-api-key", c.apiKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		SetDoNotParseResponse(true).
		Post("https://api.anthropic.com/v1/messages")

	if err != nil {
		return claude.GenerationResult{}, err
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != 200 {
		b, _ := io.ReadAll(resp.RawBody())
		return claude.GenerationResult{}, fmt.Errorf("API request failed with status code: %d and response: %s", resp.StatusCode(), string(b))
	}

	return processStreamResponse(resp.RawBody())
}

// convertMessages converts domain messages to infrastructure layer messages.
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

// processStreamResponse handles the streaming response from Claude API.
func processStreamResponse(body io.ReadCloser) (claude.GenerationResult, error) {
	reader := bufio.NewReader(body)
	var fullResponse strings.Builder
	var terminationReason string

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return claude.GenerationResult{}, err
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
			return claude.GenerationResult{}, err
		}

		if streamResp.Type == "content_block_delta" {
			fullResponse.WriteString(streamResp.Delta.Text)
		} else if streamResp.Type == "message_stop" {
			terminationReason = streamResp.Message.StopReason
		}
	}

	return claude.GenerationResult{
		Content:           fullResponse.String(),
		TerminationReason: terminationReason,
	}, nil
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
		Role       string `json:"role"`
		StopReason string `json:"stop_reason"`
	} `json:"message"`
	Delta struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
}
