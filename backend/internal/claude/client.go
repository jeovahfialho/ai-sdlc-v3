package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

const claudeAPIURL = "https://api.anthropic.com/v1/messages"

type Client struct {
	APIKey string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type ChatResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func NewClient(apiKey string) *Client {
	return &Client{APIKey: apiKey}
}

func (c *Client) GetResponse(prompt string) (string, error) {
	chatReq := ChatRequest{
		Model: "claude-3-sonnet-20240229",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 1024,
	}

	requestBody, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling request body: %v", err)
	}

	req, err := http.NewRequest("POST", claudeAPIURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling Claude API: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	log.Printf("Claude API Response: %s", string(body))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API returned non-200 status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if len(chatResp.Content) == 0 || chatResp.Content[0].Text == "" {
		return "", fmt.Errorf("no content in response")
	}

	return chatResp.Content[0].Text, nil
}
