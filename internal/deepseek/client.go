package deepseek

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// Config DeepSeek 配置
type Config struct {
	APIKey      string
	BaseURL     string
	Model       string
	FallbackModel string
	Timeout     time.Duration
	MaxRetries  int
}

// Client DeepSeek 客户端
type Client struct {
	config Config
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
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int     `json:"index"`
		Message Message `json:"message"`
		Delta   *Message `json:"delta,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// NewClient 创建 DeepSeek 客户端
func NewClient(logger *zap.Logger) (*Client, error) {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY not set")
	}

	baseURL := os.Getenv("DEEPSEEK_API_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}

	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-pro"
	}

	fallbackModel := os.Getenv("DEEPSEEK_FALLBACK_MODEL")
	if fallbackModel == "" {
		fallbackModel = "deepseek-v4-flash"
	}

	timeout := 60 * time.Second
	if t := os.Getenv("DEEPSEEK_TIMEOUT"); t != "" {
		if d, err := time.ParseDuration(t + "s"); err == nil {
			timeout = d
		}
	}

	maxRetries := 3
	if r := os.Getenv("DEEPSEEK_MAX_RETRIES"); r != "" {
		// 简单解析，省略错误处理
		fmt.Sscanf(r, "%d", &maxRetries)
	}

	return &Client{
		config: Config{
			APIKey:        apiKey,
			BaseURL:       baseURL,
			Model:         model,
			FallbackModel: fallbackModel,
			Timeout:       timeout,
			MaxRetries:    maxRetries,
		},
		logger: logger,
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// Chat 发送聊天请求
func (c *Client) Chat(messages []Message, temperature float64, maxTokens int) (*ChatResponse, error) {
	reqBody := ChatRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	resp, err := c.sendRequest(reqBody)
	if err != nil {
		// 尝试使用备用模型
		c.logger.Warn("Primary model failed, trying fallback", zap.Error(err))
		reqBody.Model = c.config.FallbackModel
		return c.sendRequest(reqBody)
	}

	return resp, nil
}

// sendRequest 发送 HTTP 请求
func (c *Client) sendRequest(reqBody ChatRequest) (*ChatResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

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

// GetModel 获取当前使用的模型
func (c *Client) GetModel() string {
	return c.config.Model
}

// SetModel 临时切换模型
func (c *Client) SetModel(model string) {
	c.config.Model = model
}
