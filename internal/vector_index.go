package internal

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
)

// ImageEmbedding 图片嵌入向量条目
type ImageEmbedding struct {
	Path      string    `json:"path"`
	Embedding []float32 `json:"embedding"`
}

// VectorIndex 向量索引（内存版）
type VectorIndex struct {
	mu        sync.RWMutex
	images    []ImageEmbedding
	indexPath string
}

// NewVectorIndex 创建新的向量索引
func NewVectorIndex(indexPath string) *VectorIndex {
	idx := &VectorIndex{
		images:    make([]ImageEmbedding, 0),
		indexPath: indexPath,
	}
	// 尝试从文件加载
	idx.Load()
	return idx
}

// Add 添加图片嵌入向量
func (vi *VectorIndex) Add(path string, embedding []float32) {
	vi.mu.Lock()
	defer vi.mu.Unlock()

	// 检查是否已存在，更新而非重复添加
	for i, img := range vi.images {
		if img.Path == path {
			vi.images[i].Embedding = embedding
			return
		}
	}

	vi.images = append(vi.images, ImageEmbedding{
		Path:      path,
		Embedding: embedding,
	})
}

// SearchByEmbedding 用嵌入向量搜索相似图片
func (vi *VectorIndex) SearchByEmbedding(queryEmbedding []float32, limit int) []SearchResult {
	vi.mu.RLock()
	defer vi.mu.RUnlock()

	if len(vi.images) == 0 {
		return nil
	}

	type scored struct {
		path  string
		score float32
	}

	scores := make([]scored, 0, len(vi.images))
	for _, img := range vi.images {
		sim := cosineSimilarity(queryEmbedding, img.Embedding)
		scores = append(scores, scored{path: img.Path, score: sim})
	}

	// 按相似度降序排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// 限制返回数量
	if limit > len(scores) {
		limit = len(scores)
	}

	results := make([]SearchResult, limit)
	for i := 0; i < limit; i++ {
		results[i] = SearchResult{
			Path:  scores[i].path,
			Score: scores[i].score,
		}
	}

	return results
}

// Size 返回索引中的图片数量
func (vi *VectorIndex) Size() int {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return len(vi.images)
}

// GetAllPaths 返回所有图片路径
func (vi *VectorIndex) GetAllPaths() []string {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	paths := make([]string, len(vi.images))
	for i, img := range vi.images {
		paths[i] = img.Path
	}
	return paths
}

// GetAllImages 获取所有图片信息
func (vi *VectorIndex) GetAllImages() []ImageEmbedding {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	result := make([]ImageEmbedding, len(vi.images))
	copy(result, vi.images)
	return result
}

// Save 保存索引到文件
func (vi *VectorIndex) Save() error {
	vi.mu.RLock()
	defer vi.mu.RUnlock()

	data, err := json.Marshal(vi.images)
	if err != nil {
		return fmt.Errorf("序列化索引失败: %w", err)
	}

	if err := os.WriteFile(vi.indexPath, data, 0644); err != nil {
		return fmt.Errorf("写入索引文件失败: %w", err)
	}

	return nil
}

// Load 从文件加载索引
func (vi *VectorIndex) Load() error {
	data, err := os.ReadFile(vi.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在不算错误
		}
		return fmt.Errorf("读取索引文件失败: %w", err)
	}

	var images []ImageEmbedding
	if err := json.Unmarshal(data, &images); err != nil {
		return fmt.Errorf("解析索引文件失败: %w", err)
	}

	vi.images = images
	return nil
}

// Clear 清空索引
func (vi *VectorIndex) Clear() {
	vi.mu.Lock()
	defer vi.mu.Unlock()
	vi.images = vi.images[:0]
}

// HasImage 检查图片是否已在索引中
func (vi *VectorIndex) HasImage(path string) bool {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	for _, img := range vi.images {
		if img.Path == path {
			return true
		}
	}
	return false
}

// SearchResult 搜索结果
type SearchResult struct {
	Path  string  `json:"path"`
	Score float32 `json:"score"`
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float32 {
	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))
	if denom == 0 {
		return 0
	}
	return dot / denom
}
