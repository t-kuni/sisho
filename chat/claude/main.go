package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"os"
	"strings"
)

type ClaudeChat struct {
	history []Message
}

func (c *ClaudeChat) addToHistory(m Message) {
	c.history = append(c.history, m)
}

type ClaudeRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	Stream      bool      `json:"stream"`
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

func (c *ClaudeChat) Send(prompt string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("Anthropic APIキーが設定されていません")
	}

	client := resty.New()

	c.addToHistory(Message{
		Role:    "user",
		Content: prompt,
	})

	requestBody := ClaudeRequest{
		Model:     "claude-3-5-sonnet-20240620",
		MaxTokens: 8192,
		Messages:  c.history,
		Stream:    true,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	resp, err := client.R().
		SetHeader("x-api-key", apiKey).
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

	reader := bufio.NewReader(resp.RawBody())
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

	responseText := fullResponse.String()
	c.addToHistory(Message{
		Role:    "assistant",
		Content: responseText,
	})

	return responseText, nil
}
