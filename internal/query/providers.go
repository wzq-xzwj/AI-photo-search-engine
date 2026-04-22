package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// =================== Ollama Provider ===================

type OllamaClient struct {
	URL   string
	Model string
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

func (c *OllamaClient) Generate(prompt string) (string, error) {
	reqBody := ollamaGenerateRequest{
		Model:  c.Model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化 Ollama 请求失败: %w", err)
	}

	url := c.URL + "/api/generate"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("调用 Ollama 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama 返回错误状态: %d", resp.StatusCode)
	}

	var result ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 Ollama 响应失败: %w", err)
	}

	return result.Response, nil
}

// =================== OpenAI-Compatible Provider ===================

type OpenAIClient struct {
	BaseURL string
	APIKey  string
	Model   string
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Generate(prompt string) (string, error) {
	reqBody := openAIChatRequest{
		Model: c.Model,
		Messages: []openAIMessage{
			{Role: "system", Content: "你是一个照片搜索查询解析助手。只输出JSON，不要任何解释。"},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化 OpenAI 请求失败: %w", err)
	}

	url := c.BaseURL + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建 OpenAI 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 OpenAI API 失败: %w", err)
	}
	defer resp.Body.Close()

	var result openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 OpenAI 响应失败: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("OpenAI API 错误: %s", result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("OpenAI 响应中没有 choices")
	}

	return result.Choices[0].Message.Content, nil
}
