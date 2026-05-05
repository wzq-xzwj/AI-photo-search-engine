package handlers

import (
	"net/http"
	"strconv"

	"photo-search-engine/internal"
	"photo-search-engine/internal/db"
	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

import (
	"encoding/json"
	"os"
)

var similarityLabels = loadLabelsV2()

func loadLabelsV2() []string {
	v2Path := "config/label_space_v2.json"
	if envPath := os.Getenv("LABEL_SPACE_PATH"); envPath != "" {
		v2Path = envPath
	}
	if data, err := os.ReadFile(v2Path); err == nil {
		var spaceV2 struct {
			Categories map[string][]string `json:"categories"`
		}
		if json.Unmarshal(data, &spaceV2) == nil {
			var flat []string
			for _, labels := range spaceV2.Categories {
				flat = append(flat, labels...)
			}
			if len(flat) > 0 {
				return flat
			}
		}
	}
	// fallback
	return []string{"自然风光", "人物合影", "城市街景", "古建筑", "美食餐饮", "花卉"}
}

type PhotosHandler struct {
	indexer  *indexer.Index
	mlClient *internal.MLClient
	db       *db.DB
	logger   *zap.Logger
}

func NewPhotosHandler(idx *indexer.Index, mlClient *internal.MLClient, database *db.DB, logger *zap.Logger) *PhotosHandler {
	return &PhotosHandler{indexer: idx, mlClient: mlClient, db: database, logger: logger}
}

func (h *PhotosHandler) HandleList(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "40")
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
				// 使用 V2 分类器（Prompt模板 + 阈值过滤），失败时回退 V1
			results, err := h.mlClient.SimilarityBatchV2(batch, similarityLabels, 10, 0.25)
			if err != nil {
				h.logger.Warn("V2 分类失败，回退 V1", zap.Error(err))
				results, err = h.mlClient.SimilarityBatch(batch, similarityLabels, 3)
				if err != nil {
					h.logger.Warn("ClassifyBatch failed", zap.Error(err), zap.Int("batch", i/batchSize))
					continue
				}
			}
			updates := make(map[string][]string)
			for j, result := range results {
				path := batch[j]
				mlTags := make([]string, 0, len(result))
				for _, item := range result {
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
			// 同步保存到数据库
			if h.db != nil {
				for path, tags := range updates {
					photos := h.indexer.ListAllPhotos()
					for _, p := range photos {
						if p.Path == path {
							h.db.UpdatePhotoTags(p.ID, tags)
							break
						}
					}
				}
			}
			h.logger.Info("ClassifyBatch completed", zap.Int("batch", i/batchSize), zap.Int("count", len(batch)))
		}
	}(allPaths, existingTagsMap)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Classification started",
		"total":   len(allPaths),
	})
}
