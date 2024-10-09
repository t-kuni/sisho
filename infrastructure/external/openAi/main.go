package openAi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	domainOpenAI "github.com/t-kuni/sisho/domain/external/openAi"
	"io"
	"os"
	"strings"
)

const apiURL = "https://api.openai.com/v1/chat/completions"

type OpenAIClient struct {
	httpClient *resty.Client
}

type apiRequest struct {
	Model    string           `json:"model"`
	Messages []apiMessageItem `json:"messages"`
	Stream   bool             `json:"stream"`
}

type apiMessageItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func NewOpenAIClient() *OpenAIClient {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	return &OpenAIClient{
		httpClient: client,
	}
}

func (c *OpenAIClient) SendMessage(messages []domainOpenAI.Message, model string) (domainOpenAI.GenerationResult, error) {
	apiMessages := make([]apiMessageItem, len(messages))
	for i, msg := range messages {
		apiMessages[i] = apiMessageItem{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	reqBody := apiRequest{
		Model:    model,
		Messages: apiMessages,
		Stream:   true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return domainOpenAI.GenerationResult{}, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := c.httpClient.R().
		SetBody(jsonBody).
		SetDoNotParseResponse(true).
		Post(apiURL)

	if err != nil {
		return domainOpenAI.GenerationResult{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != 200 {
		bodyBytes, _ := io.ReadAll(resp.RawBody())
		return domainOpenAI.GenerationResult{}, fmt.Errorf("API request failed with status code %d and response: %s", resp.StatusCode(), string(bodyBytes))
	}

	return processStreamResponse(resp.RawBody())
}

func processStreamResponse(body io.ReadCloser) (domainOpenAI.GenerationResult, error) {
	reader := bufio.NewReader(body)
	var fullResponse strings.Builder
	var terminationReason string

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return domainOpenAI.GenerationResult{}, fmt.Errorf("error reading stream: %w", err)
		}

		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data: ")) {
			continue
		}

		data := bytes.TrimPrefix(line, []byte("data: "))
		if string(data) == "[DONE]" {
			break
		}

		var streamResp apiResponse
		if err := json.Unmarshal(data, &streamResp); err != nil {
			return domainOpenAI.GenerationResult{}, fmt.Errorf("failed to unmarshal stream data: %w", err)
		}

		if len(streamResp.Choices) > 0 {
			fullResponse.WriteString(streamResp.Choices[0].Delta.Content)
			if streamResp.Choices[0].FinishReason != "" {
				terminationReason = streamResp.Choices[0].FinishReason
			}
		}
	}

	return domainOpenAI.GenerationResult{
		Content:           fullResponse.String(),
		TerminationReason: terminationReason,
	}, nil
}
