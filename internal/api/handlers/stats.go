package handlers

import (
	"net/http"

	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StatsHandler struct {
	indexer *indexer.Index
	logger  *zap.Logger
}

func NewStatsHandler(idx *indexer.Index, logger *zap.Logger) *StatsHandler {
	return &StatsHandler{indexer: idx, logger: logger}
}

func (h *StatsHandler) HandleStats(c *gin.Context) {
	stats := h.indexer.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"total_photos":  stats.TotalPhotos,
		"total_tags":    stats.TotalTags,
		"total_size":    stats.TotalSize,
		"vector_images": stats.VectorImages,
	})
}
