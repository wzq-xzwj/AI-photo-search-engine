package handlers

import (
	"net/http"
	"strconv"

	"photo-search-engine/internal"
	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

import (
	"encoding/json"
	"os"
)

var similarityLabels = loadSimilarityLabels()

func loadSimilarityLabels() []string {
	path := "config/label_space_v1.json"
	if envPath := os.Getenv("LABEL_SPACE_PATH"); envPath != "" {
		path = envPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{
			"城市街景", "古建筑", "现代建筑", "公园园林", "海滨沙滩",
			"山川湖泊", "室内场景", "夜景灯光", "人物合影", "美食餐饮",
		}
	}
	var space struct {
		Flat []string `json:"flat"`
	}
	if err := json.Unmarshal(data, &space); err != nil {
		return []string{"城市街景", "古建筑", "室内场景", "人物合影", "美食餐饮"}
	}
	if len(space.Flat) == 0 {
		return []string{"城市街景", "古建筑", "室内场景", "人物合影", "美食餐饮"}
	}
	return space.Flat
}

type PhotosHandler struct {
	indexer  *indexer.Index
	mlClient *internal.MLClient
	logger   *zap.Logger
}

func NewPhotosHandler(idx *indexer.Index, mlClient *internal.MLClient, logger *zap.Logger) *PhotosHandler {
	return &PhotosHandler{indexer: idx, mlClient: mlClient, logger: logger}
}

func (h *PhotosHandler) HandleList(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")
	dir := c.Query("dir")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	var photos []indexer.Photo
	var total int
	var err error

	if dir != "" {
		photos, total, err = h.indexer.ListPhotosByDir(dir, page, pageSize)
	} else {
		photos, total, err = h.indexer.ListPhotos(page, pageSize)
	}

	if err != nil {
		h.logger.Error("Failed to list photos", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list photos"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"photos":    photos,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (h *PhotosHandler) HandleGet(c *gin.Context) {
	id := c.Param("id")

	photo, err := h.indexer.GetPhoto(id)
	if err != nil {
		h.logger.Error("Failed to get photo", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Photo not found"})
		return
	}

	c.JSON(http.StatusOK, photo)
}

func (h *PhotosHandler) HandleClassifyAll(c *gin.Context) {
	if h.mlClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ML client not available"})
		return
	}

	// 获取所有照片路径和现有标签
	page := 1
	pageSize := 1000
	var allPaths []string
	existingTagsMap := make(map[string][]string)
	for {
		photos, total, err := h.indexer.ListPhotos(page, pageSize)
		if err != nil {
			h.logger.Error("Failed to list photos for classification", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list photos"})
			return
		}
		for _, p := range photos {
			allPaths = append(allPaths, p.Path)
			existingTagsMap[p.Path] = p.Tags
		}
		if len(photos) < pageSize || len(allPaths) >= total {
			break
		}
		page++
	}

	if len(allPaths) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No photos to classify", "updated": 0})
		return
	}

	// 异步批量分类，避免阻塞 HTTP 响应
	go func(paths []string, tagsMap map[string][]string) {
		batchSize := 10
		for i := 0; i < len(paths); i += batchSize {
			end := i + batchSize
			if end > len(paths) {
				end = len(paths)
			}
			batch := paths[i:end]
			results, err := h.mlClient.SimilarityBatch(batch, similarityLabels, 3)
			if err != nil {
				h.logger.Warn("ClassifyBatch failed", zap.Error(err), zap.Int("batch", i/batchSize))
				continue
			}
			updates := make(map[string][]string)
			for j, result := range results {
				path := batch[j]
				mlTags := make([]string, 0, 3)
				for idx, item := range result {
					if idx >= 3 {
						break
					}
					mlTags = append(mlTags, item.Label)
				}
				// 与现有标签合并（保留文件夹标签，去重）
				existingTags := tagsMap[path]
				tagSet := make(map[string]bool)
				var merged []string
				for _, t := range existingTags {
					if !tagSet[t] {
						tagSet[t] = true
						merged = append(merged, t)
					}
				}
				for _, t := range mlTags {
					if !tagSet[t] {
						tagSet[t] = true
						merged = append(merged, t)
					}
				}
				updates[path] = merged
			}
			h.indexer.BatchUpdateTagsByPaths(updates)
			h.logger.Info("ClassifyBatch completed", zap.Int("batch", i/batchSize), zap.Int("count", len(batch)))
		}
	}(allPaths, existingTagsMap)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Classification started",
		"total":   len(allPaths),
	})
}
