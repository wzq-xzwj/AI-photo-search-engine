package router

import (
	"photo-search-engine/internal"
	"photo-search-engine/internal/api/handlers"
	"photo-search-engine/internal/indexer"
	"photo-search-engine/internal/scanner"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	indexer   *indexer.Index
	scanner   *scanner.Scanner
	mlClient  *internal.MLClient
	logger    *zap.Logger
}

func NewRouter(idx *indexer.Index, scannerSvc *scanner.Scanner, mlClient *internal.MLClient, logger *zap.Logger) *Router {
	return &Router{
		indexer:  idx,
		scanner:  scannerSvc,
		mlClient: mlClient,
		logger:   logger,
	}
}

func (r *Router) Setup() *gin.Engine {
	engine := gin.Default()

	api := engine.Group("/api/v1")
	{
		searchHandler := handlers.NewSearchHandler(r.indexer, r.logger)
		api.GET("/search", searchHandler.HandleSearch)

		photosHandler := handlers.NewPhotosHandler(r.indexer, r.mlClient, r.logger)
		api.GET("/photos", photosHandler.HandleList)
		api.GET("/photos/:id", photosHandler.HandleGet)

		chatHandler := handlers.NewChatHandler(r.indexer, r.logger)
		api.POST("/chat", chatHandler.HandleChat)

		statsHandler := handlers.NewStatsHandler(r.indexer, r.logger)
		api.GET("/stats", statsHandler.HandleStats)

		filesHandler := handlers.NewFilesHandler(r.scanner, r.logger)
		api.POST("/files/scan", filesHandler.HandleScan)
	}

	return engine
}
