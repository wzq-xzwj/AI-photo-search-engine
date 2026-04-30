package internal

import (
	"fmt"
	"net"
	"os"
	"time"

	"photo-search-engine/internal/vector"
)

// VectorIndex 基于 Milvus 的向量索引
type VectorIndex struct {
	milvus *vector.MilvusIndex
}

// NewVectorIndex 创建新的向量索引
func NewVectorIndex(indexPath string) *VectorIndex {
	// 快速检查 Milvus 是否可达
	milvusAddr := os.Getenv("MILVUS_ADDRESS")
	if milvusAddr == "" {
		milvusAddr = "localhost:19530"
	}
	
	conn, err := net.DialTimeout("tcp", milvusAddr, 2*time.Second)
	if err != nil {
		fmt.Printf("Milvus 不可达 (%s): %v，跳过向量索引\n", milvusAddr, err)
		return nil
	}
	conn.Close()

	milvus, err := vector.NewMilvusIndex(milvusAddr)
	if err != nil {
		fmt.Printf("初始化 Milvus 失败: %v\n", err)
		return nil
	}

	return &VectorIndex{
		milvus: milvus,
	}
}

// Add 添加图片嵌入向量
func (vi *VectorIndex) Add(path string, embedding []float32) {
	if vi == nil || vi.milvus == nil {
		return
	}

	photoID := path // 使用路径作为 photoID
	if err := vi.milvus.AddPhoto(photoID, path, embedding); err != nil {
		fmt.Printf("添加向量失败: %v\n", err)
	}
}

// BatchAdd 批量添加图片嵌入向量
func (vi *VectorIndex) BatchAdd(items []ImageEmbedding) {
	if vi == nil || vi.milvus == nil {
		return
	}

	photoIDs := make([]string, len(items))
	filePaths := make([]string, len(items))
	vectors := make([][]float32, len(items))

	for i, item := range items {
		photoIDs[i] = item.Path
		filePaths[i] = item.Path
		vectors[i] = item.Embedding
	}

	if err := vi.milvus.AddPhotosBatch(photoIDs, filePaths, vectors); err != nil {
		fmt.Printf("批量添加向量失败: %v\n", err)
	}
}

// SearchByEmbedding 用嵌入向量搜索相似图片
func (vi *VectorIndex) SearchByEmbedding(queryEmbedding []float32, limit int) []SearchResult {
	if vi == nil || vi.milvus == nil {
		return nil
	}

	photoIDs, filePaths, scores, err := vi.milvus.Search(queryEmbedding, limit)
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
		return nil
	}

	results := make([]SearchResult, len(photoIDs))
	for i := range photoIDs {
		results[i] = SearchResult{
			Path:  filePaths[i],
			Score: scores[i],
		}
	}

	return results
}

// Size 返回索引中的图片数量
func (vi *VectorIndex) Size() int {
	if vi == nil || vi.milvus == nil {
		return 0
	}

	count, err := vi.milvus.GetStats()
	if err != nil {
		return 0
	}
	return int(count)
}

// GetAllPaths 返回所有图片路径
func (vi *VectorIndex) GetAllPaths() []string {
	// Milvus 不直接支持获取所有路径，需要从 SQLite 获取
	return nil
}

// GetAllImages 获取所有图片信息
func (vi *VectorIndex) GetAllImages() []ImageEmbedding {
	// Milvus 不直接支持获取所有数据，需要从 SQLite 获取
	return nil
}

// Save 保存索引（Milvus 自动持久化，无需手动保存）
func (vi *VectorIndex) Save() error {
	// Milvus 自动持久化
	return nil
}

// Load 加载索引（Milvus 自动加载）
func (vi *VectorIndex) Load() error {
	// Milvus 自动加载
	return nil
}

// Clear 清空索引
func (vi *VectorIndex) Clear() {
	if vi == nil || vi.milvus == nil {
		return
	}

	if err := vi.milvus.DropCollection(); err != nil {
		fmt.Printf("清空索引失败: %v\n", err)
	}
}

// HasImage 检查图片是否已在索引中
func (vi *VectorIndex) HasImage(path string) bool {
	if vi == nil || vi.milvus == nil {
		return false
	}
	return vi.milvus.HasPhoto(path)
}

// Close 关闭连接
func (vi *VectorIndex) Close() error {
	if vi == nil || vi.milvus == nil {
		return nil
	}
	return vi.milvus.Close()
}

// ImageEmbedding 图片嵌入向量条目
type ImageEmbedding struct {
	Path      string    `json:"path"`
	Embedding []float32 `json:"embedding"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Path  string  `json:"path"`
	Score float32 `json:"score"`
}
