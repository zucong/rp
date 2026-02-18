package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zucong/rp/models"
)

type Client struct {
	config *models.Config
}

func NewClient(cfg *models.Config) *Client {
	return &Client{config: cfg}
}

func (c *Client) UpdateConfig(cfg *models.Config) {
	c.config = cfg
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MergeSystemPrompts combines multiple system prompts into one for API compatibility
// Some LLM APIs (Claude, etc.) only support a single system message
func MergeSystemPrompts(prompts []string) string {
	var result strings.Builder
	for i, p := range prompts {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if i > 0 {
			result.WriteString("\n\n---\n\n")
		}
		result.WriteString(strings.TrimSpace(p))
	}
	return result.String()
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message      *Message `json:"message,omitempty"`
		Delta        *Message `json:"delta,omitempty"`
		FinishReason string   `json:"finish_reason"`
	} `json:"choices"`
}

func (c *Client) Complete(messages []Message, model string, temperature float64, maxTokens int) (string, error) {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.config.APIEndpoint+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error: %s", string(body))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 || result.Choices[0].Message == nil {
		return "", fmt.Errorf("no response from API")
	}

	return result.Choices[0].Message.Content, nil
}

func (c *Client) StreamComplete(messages []Message, model string, temperature float64, maxTokens int, onToken func(string)) error {
	reqBody := ChatRequest{
		Model:       model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.config.APIEndpoint+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s", string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !bytes.HasPrefix([]byte(line), []byte("data: ")) {
			continue
		}
		data := bytes.TrimPrefix([]byte(line), []byte("data: "))
		if string(data) == "[DONE]" {
			break
		}

		var streamResp ChatResponse
		if err := json.Unmarshal(data, &streamResp); err != nil {
			continue
		}

		if len(streamResp.Choices) > 0 && streamResp.Choices[0].Delta != nil {
			onToken(streamResp.Choices[0].Delta.Content)
		}
	}

	return scanner.Err()
}
