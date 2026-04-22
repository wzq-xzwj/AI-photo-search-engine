package handlers

import (
	"net/http"

	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DirsHandler struct {
	indexer *indexer.Index
	logger  *zap.Logger
}

func NewDirsHandler(idx *indexer.Index, logger *zap.Logger) *DirsHandler {
	return &DirsHandler{indexer: idx, logger: logger}
}

func (h *DirsHandler) HandleList(c *gin.Context) {
	dirs := h.indexer.ListDirs()
	c.JSON(http.StatusOK, gin.H{
		"dirs": dirs,
	})
}
