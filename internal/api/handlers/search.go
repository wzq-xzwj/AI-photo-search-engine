package handlers

import (
	"net/http"
	"strconv"

	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SearchHandler struct {
	indexer *indexer.Index
	logger  *zap.Logger
}

func NewSearchHandler(idx *indexer.Index, logger *zap.Logger) *SearchHandler {
	return &SearchHandler{indexer: idx, logger: logger}
}

func (h *SearchHandler) HandleSearch(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)

	results, err := h.indexer.Search(query, limit)
	if err != nil {
		h.logger.Error("Search failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"query":   query,
		"results": results,
		"total":   len(results),
	})
}
