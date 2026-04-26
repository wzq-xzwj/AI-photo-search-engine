package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	MLServiceURL = "http://127.0.0.1:8000"
	EmbeddingDim = 512
)

// MLClient ML 服务 HTTP 客户端
type MLClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMLClient 创建新的 ML 客户端（带连接池）
func NewMLClient(baseURL string) *MLClient {
	if baseURL == "" {
		baseURL = MLServiceURL
	}

	// 配置连接池
	transport := &http.Transport{
		// 连接池设置
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20,
		// 空闲连接超时
		IdleConnTimeout: 90 * time.Second,
		// TLS 握手超时
		TLSHandshakeTimeout: 10 * time.Second,
		// 连接超时
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// 禁用压缩（ML服务通常返回小数据）
		DisableCompression: false,
		// 强制尝试 HTTP/2
		ForceAttemptHTTP2: true,
	}

	return &MLClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second, // 增加超时以支持批量请求
		},
	}
}

// EmbeddingResponse ML 服务返回的嵌入向量
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
	Dim       int       `json:"dim"`
}

// ClassificationItem 单条分类结果
type ClassificationItem struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

// ClassifyBatchResponse 批量分类响应
type ClassifyBatchResponse struct {
	Results [][]ClassificationItem `json:"results"`
}

// TextEmbedRequest 文本编码请求
type TextEmbedRequest struct {
	Text string `json:"text"`
}

// ExtractImageEmbedding 调用 ML 服务提取图像特征
func (c *MLClient) ExtractImageEmbedding(imagePath string) ([]float32, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return nil, fmt.Errorf("打开图片文件失败: %w", err)
	}
	defer file.Close()

	// 构建 multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 获取正确的 MIME 类型
	ext := filepath.Ext(imagePath)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "image/jpeg" // 默认
	}

	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename="%s"`, filepath.Base(imagePath))}
	h["Content-Type"] = []string{mimeType}
	part, err := writer.CreatePart(textproto.MIMEHeader(h))
	if err != nil {
		return nil, fmt.Errorf("创建 form 文件失败: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("复制文件内容失败: %w", err)
	}

	writer.Close()

	// 发送请求
	url := fmt.Sprintf("%s/api/v1/extract", c.baseURL)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 ML 服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML 服务返回错误状态 %d: %s", resp.StatusCode, string(respBody))
	}

	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 ML 响应失败: %w", err)
	}

	if len(result.Embedding) != EmbeddingDim {
		return nil, fmt.Errorf("嵌入向量维度不匹配: 期望 %d, 实际 %d", EmbeddingDim, len(result.Embedding))
	}

	return result.Embedding, nil
}

// EncodeText 调用 ML 服务编码文本
func (c *MLClient) EncodeText(text string) ([]float32, error) {
	reqBody := TextEmbedRequest{Text: text}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/embed/text", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 ML 服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML 服务返回错误状态 %d: %s", resp.StatusCode, string(respBody))
	}

	var result EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 ML 响应失败: %w", err)
	}

	if len(result.Embedding) != EmbeddingDim {
		return nil, fmt.Errorf("嵌入向量维度不匹配: 期望 %d, 实际 %d", EmbeddingDim, len(result.Embedding))
	}

	return result.Embedding, nil
}

// ClassifyBatch 批量分类图片
func (c *MLClient) ClassifyBatch(imagePaths []string, topK int) ([][]ClassificationItem, error) {
	if len(imagePaths) == 0 {
		return nil, nil
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, imagePath := range imagePaths {
		file, err := os.Open(imagePath)
		if err != nil {
			return nil, fmt.Errorf("打开图片文件失败: %w", err)
		}
		defer file.Close()

		ext := filepath.Ext(imagePath)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "image/jpeg"
		}

		h := make(map[string][]string)
		h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="files"; filename="%s"`, filepath.Base(imagePath))}
		h["Content-Type"] = []string{mimeType}
		part, err := writer.CreatePart(textproto.MIMEHeader(h))
		if err != nil {
			return nil, fmt.Errorf("创建 form 文件失败: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("复制文件内容失败: %w", err)
		}
	}
	writer.Close()

	url := fmt.Sprintf("%s/api/v1/classify/batch?top_k=%d", c.baseURL, topK)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 ML 服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML 服务返回错误状态 %d: %s", resp.StatusCode, string(respBody))
	}

	var result ClassifyBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 ML 响应失败: %w", err)
	}
	return result.Results, nil
}

// SimilarityBatchResponse raw similarity response
type SimilarityBatchResponse struct {
	Results [][]ClassificationItem `json:"results"`
}

// SimilarityBatch 批量计算图片与标签的原始 cosine similarity（无 softmax）
func (c *MLClient) SimilarityBatch(imagePaths []string, labels []string, topK int) ([][]ClassificationItem, error) {
	if len(imagePaths) == 0 {
		return nil, nil
	}
	if len(labels) == 0 {
		return nil, fmt.Errorf("labels cannot be empty")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// 写入 files
	for _, imagePath := range imagePaths {
		file, err := os.Open(imagePath)
		if err != nil {
			return nil, fmt.Errorf("打开图片文件失败: %w", err)
		}
		defer file.Close()

		ext := filepath.Ext(imagePath)
		mimeType := mime.TypeByExtension(ext)
		if mimeType == "" {
			mimeType = "image/jpeg"
		}

		h := make(map[string][]string)
		h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="files"; filename="%s"`, filepath.Base(imagePath))}
		h["Content-Type"] = []string{mimeType}
		part, err := writer.CreatePart(textproto.MIMEHeader(h))
		if err != nil {
			return nil, fmt.Errorf("创建 form 文件失败: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			return nil, fmt.Errorf("复制文件内容失败: %w", err)
		}
	}

	// 写入 labels 和 top_k
	labelsJSON, _ := json.Marshal(labels)
	writer.WriteField("labels", string(labelsJSON))
	writer.WriteField("top_k", strconv.Itoa(topK))
	writer.Close()

	url := fmt.Sprintf("%s/api/v1/similarity/batch", c.baseURL)
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 ML 服务失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ML 服务返回错误状态 %d: %s", resp.StatusCode, string(respBody))
	}

	var result SimilarityBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 ML 响应失败: %w", err)
	}
	return result.Results, nil
}

// HealthCheck 检查 ML 服务是否可用
func (c *MLClient) HealthCheck() error {
	url := fmt.Sprintf("%s/api/v1/health", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("ML 服务不可达: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ML 服务状态异常: %d", resp.StatusCode)
	}
	return nil
}

// ChatRequest ml-service chat 请求

type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

// ChatResponse ml-service chat 响应

type ChatResponse struct {
	Response string `json:"response"`
}

// FaceInfo 人脸信息
type FaceInfo struct {
	Index      int       `json:"index"`
	Location   FaceLoc   `json:"location"`
	Encoding   []float64 `json:"encoding"`
	Confidence float64   `json:"confidence"`
}

// FaceLoc 人脸位置
type FaceLoc struct {
	Top    int `json:"top"`
	Right  int `json:"right"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
}

// DetectFacesResponse 人脸检测响应
type DetectFacesResponse struct {
	ImagePath string     `json:"image_path"`
	FaceCount int        `json:"face_count"`
	Faces     []FaceInfo `json:"faces"`
}

// DetectFaces 调用 ML 服务检测照片中的人脸
func (c *MLClient) DetectFaces(imagePath string) (*DetectFacesResponse, error) {
	type faceReq struct {
		ImagePath string `json:"image_path"`
	}
	jsonData, _ := json.Marshal(faceReq{ImagePath: imagePath})

	url := fmt.Sprintf("%s/api/v1/faces/detect", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建人脸检测请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用人脸检测失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 503 {
		return nil, fmt.Errorf("face detection not available")
	}
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("人脸检测返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	var result DetectFacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析人脸检测响应失败: %w", err)
	}
	return &result, nil
}

// Chat 调用 ML 服务的 chat API
func (c *MLClient) Chat(message string) (string, error) {
	reqBody := ChatRequest{SessionID: "default", Message: message}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化 chat 请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/chat", c.baseURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建 chat 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 ML chat 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ML chat 返回错误 %d: %s", resp.StatusCode, string(respBody))
	}

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析 chat 响应失败: %w", err)
	}

	return result.Response, nil
}
