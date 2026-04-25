package deepseek

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"photo-search-engine/internal/config"
	"go.uber.org/zap"
)

// Client DeepSeek 客户端
type Client struct {
	cfg    *config.Config
	logger *zap.Logger
	client *http.Client
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient 创建 DeepSeek 客户端
func NewClient(logger *zap.Logger) (*Client, error) {
	cfg := config.Get()
	
	if cfg.LLMAPIKey == "" {
		return nil, fmt.Errorf("LLM API key not configured, set DEEPSEEK_API_KEY")
	}

	return &Client{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

// Chat 发送聊天请求
func (c *Client) Chat(messages []Message, temperature float64, maxTokens int) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:       c.cfg.LLMModel,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	url := fmt.Sprintf("%s/chat/completions", c.cfg.LLMBaseURL)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.cfg.LLMAPIKey))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &chatResp, nil
}
