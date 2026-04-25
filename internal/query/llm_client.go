package query

import (
	"fmt"
	"os"
)

// LLMClient 抽象不同大模型提供商的调用接口
// 统一使用 OpenAI 兼容协议，支持：
//   - DeepSeek:   OPENAI_BASE_URL=https://api.deepseek.com/v1
//   - Ollama:     OPENAI_BASE_URL=http://127.0.0.1:11434/v1  OPENAI_API_KEY=ollama
//   - OpenAI:     OPENAI_BASE_URL=https://api.openai.com/v1
//   - 以及其他所有 OpenAI 兼容接口
type LLMClient interface {
	Generate(prompt string) (string, error)
}

// NewLLMClientFromEnv 根据环境变量创建 OpenAI 兼容客户端
func NewLLMClientFromEnv() (LLMClient, error) {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1"
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("环境变量 OPENAI_API_KEY 未设置")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "deepseek-chat"
	}
	return &OpenAIClient{BaseURL: baseURL, APIKey: apiKey, Model: model}, nil
}
