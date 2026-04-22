package query

import (
	"fmt"
	"os"
)

// LLMClient 抽象不同大模型提供商的调用接口
type LLMClient interface {
	Generate(prompt string) (string, error)
}

// NewLLMClientFromEnv 根据环境变量创建对应的 LLM 客户端
func NewLLMClientFromEnv() (LLMClient, error) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}

	switch provider {
	case "ollama":
		url := os.Getenv("OLLAMA_URL")
		if url == "" {
			url = "http://127.0.0.1:11434"
		}
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "gemma3:4b"
		}
		return &OllamaClient{URL: url, Model: model}, nil

	case "openai", "compatible":
		baseURL := os.Getenv("OPENAI_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.openai.com/v1"
		}
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("环境变量 OPENAI_API_KEY 未设置")
		}
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			model = "gpt-4o-mini"
		}
		return &OpenAIClient{BaseURL: baseURL, APIKey: apiKey, Model: model}, nil

	default:
		return nil, fmt.Errorf("不支持的 LLM_PROVIDER: %s", provider)
	}
}
