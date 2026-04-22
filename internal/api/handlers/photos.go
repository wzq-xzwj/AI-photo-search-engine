package handlers

import (
	"net/http"
	"strconv"

	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PhotosHandler struct {
	indexer *indexer.Index
	logger  *zap.Logger
}

func NewPhotosHandler(idx *indexer.Index, logger *zap.Logger) *PhotosHandler {
	return &PhotosHandler{indexer: idx, logger: logger}
}

func (h *PhotosHandler) HandleList(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)

	photos, total, err := h.indexer.ListPhotos(page, pageSize)
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
