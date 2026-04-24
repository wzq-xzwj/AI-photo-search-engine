package router

import (
	"photo-search-engine/internal"
	"photo-search-engine/internal/api/handlers"
	"photo-search-engine/internal/db"
	"photo-search-engine/internal/indexer"
	"photo-search-engine/internal/scanner"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Router struct {
	indexer   *indexer.Index
	scanner   *scanner.Scanner
	mlClient  *internal.MLClient
	db        *db.DB
	logger    *zap.Logger
}

func NewRouter(idx *indexer.Index, scannerSvc *scanner.Scanner, mlClient *internal.MLClient, database *db.DB, logger *zap.Logger) *Router {
	return &Router{
		indexer:  idx,
		scanner:  scannerSvc,
		mlClient: mlClient,
		db:       database,
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

		// Face recognition routes
		if r.db != nil {
			faceHandler := handlers.NewFaceHandler(r.indexer, r.db, "", r.logger)
			api.POST("/faces/detect", faceHandler.HandleDetectFaces)
			api.PUT("/faces/:id/label", faceHandler.HandleLabelFace)
			api.GET("/persons", faceHandler.HandleGetPersons)
			api.GET("/search/person", faceHandler.HandleSearchByPerson)
			api.GET("/faces/thumbnail", faceHandler.HandleGetFaceThumbnail)
		}
	}

	return engine
}
