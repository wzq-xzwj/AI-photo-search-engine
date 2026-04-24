package deepseek

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// LLMService DeepSeek LLM 服务
type LLMService struct {
	client *Client
	logger *zap.Logger
}

// NewLLMService 创建 LLM 服务
func NewLLMService(logger *zap.Logger) (*LLMService, error) {
	client, err := NewClient(logger)
	if err != nil {
		return nil, err
	}

	return &LLMService{
		client: client,
		logger: logger,
	}, nil
}

// AnalyzePhoto 分析照片内容
func (s *LLMService) AnalyzePhoto(ctx context.Context, description string) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个专业的照片分析助手。请根据照片描述，分析照片中的场景、人物、物品、氛围等信息，用中文简洁回答。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("请分析这张照片：%s", description),
		},
	}

	resp, err := s.client.Chat(messages, 0.7, 500)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateTags 生成照片标签
func (s *LLMService) GenerateTags(ctx context.Context, description string) ([]string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个照片标签生成助手。请根据照片描述，生成5-10个相关的中文标签，用逗号分隔。标签应该简洁、具体、有代表性。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("请为这张照片生成标签：%s", description),
		},
	}

	resp, err := s.client.Chat(messages, 0.5, 200)
	if err != nil {
		return nil, err
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// 解析标签
	content := resp.Choices[0].Message.Content
	tags := strings.Split(content, ",")
	
	// 清理标签
	var cleanTags []string
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleanTags = append(cleanTags, tag)
		}
	}

	return cleanTags, nil
}

// AnswerQuestion 回答关于照片的问题
func (s *LLMService) AnswerQuestion(ctx context.Context, description string, question string) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个照片问答助手。请根据照片描述，回答用户的问题。如果无法从描述中得出答案，请诚实说明。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("照片描述：%s\n\n问题：%s", description, question),
		},
	}

	resp, err := s.client.Chat(messages, 0.7, 500)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}

// SearchQuery 优化搜索查询
func (s *LLMService) SearchQuery(ctx context.Context, query string) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个搜索查询优化助手。请将用户的自然语言查询转换为更精确的搜索关键词，用于照片搜索引擎。保持中文，提取关键元素。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("优化这个搜索查询：%s", query),
		},
	}

	resp, err := s.client.Chat(messages, 0.3, 200)
	if err != nil {
		return query, err // 失败时返回原查询
	}

	if len(resp.Choices) == 0 {
		return query, nil
	}

	return resp.Choices[0].Message.Content, nil
}

// FaceDescription 描述人脸特征
func (s *LLMService) FaceDescription(ctx context.Context, faceInfo string) (string, error) {
	messages := []Message{
		{
			Role:    "system",
			Content: "你是一个人脸识别助手。请根据人脸检测信息，描述这个人的特征（性别、年龄、表情、是否戴眼镜等）。",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("描述这个人脸：%s", faceInfo),
		},
	}

	resp, err := s.client.Chat(messages, 0.5, 300)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return resp.Choices[0].Message.Content, nil
}
