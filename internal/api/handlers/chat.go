package handlers

import (
	"net/http"

	"photo-search-engine/internal/indexer"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ChatHandler struct {
	indexer *indexer.Index
	logger  *zap.Logger
}

func NewChatHandler(idx *indexer.Index, logger *zap.Logger) *ChatHandler {
	return &ChatHandler{indexer: idx, logger: logger}
}

type ChatRequest struct {
	Message string `json:"message" binding:"required"`
}

func (h *ChatHandler) HandleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	response, err := h.indexer.Chat(req.Message)
	if err != nil {
		h.logger.Error("Chat failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"response": response,
	})
}
