package main

import (
	"log"
	"net/http"
	"os"

	"photo-search-engine/internal"
	"photo-search-engine/internal/api/handlers"
	"photo-search-engine/internal/api/middleware"
	"photo-search-engine/internal/config"
	"photo-search-engine/internal/db"
	"photo-search-engine/internal/indexer"
	"photo-search-engine/internal/query"
	"photo-search-engine/internal/scanner"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	// Load .env file if exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// 加载统一配置
	cfg := config.Get()

	// Initialize logger
	var zapLog *zap.Logger
	var err error
	switch cfg.LogLevel {
	case "debug":
		zapLog, err = zap.NewDevelopment()
	default:
		zapLog, err = zap.NewProduction()
	}
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLog.Sync()

	// 初始化数据库
	database, err := db.New(cfg.DBPath)
	if err != nil {
		zapLog.Warn("Database initialization failed", zap.Error(err))
	} else {
		zapLog.Info("Database initialized", zap.String("path", cfg.DBPath))
		defer database.Close()
	}

	// 尝试从 JSON 迁移数据（仅首次）
	if database != nil {
		count, _ := database.GetPhotoCount()
		if count == 0 {
			if err := database.MigrateFromJSON("photo_tags.json", "photo_tags.json"); err != nil {
				zapLog.Warn("JSON migration failed", zap.Error(err))
			}
		}
	}

	// 初始化 ML 客户端
	mlClient := internal.NewMLClient(cfg.MLServiceURL)
	zapLog.Info("ML client initialized", zap.String("url", cfg.MLServiceURL))

	// 检查 ML 服务是否可用
	if err := mlClient.HealthCheck(); err != nil {
		zapLog.Warn("ML service not available, semantic search disabled", zap.Error(err))
	} else {
		zapLog.Info("ML service is available")
	}

	// 初始化向量索引
	vectorIndex := internal.NewVectorIndex(cfg.VectorIndexPath)
	zapLog.Info("Vector index initialized", zap.Int("images", vectorIndex.Size()), zap.String("path", cfg.VectorIndexPath))

	// Initialize components with ML client
	idx := indexer.NewWithML(mlClient, vectorIndex)

	// 创建带数据库的扫描器
	var scanSvc *scanner.Scanner
	if database != nil {
		scanSvc = scanner.NewWithDB(idx, mlClient, vectorIndex, database)
	} else {
		scanSvc = scanner.NewWithML(idx, mlClient, vectorIndex)
	}

	// 从数据库加载照片数据（优先）
	if database != nil {
		photos, err := database.LoadPhotos()
		if err == nil && len(photos) > 0 {
			zapLog.Info("从数据库恢复照片数据", zap.Int("count", len(photos)))
			idx.UpdatePhotosMeta(photos)
		}
	}

	// 从向量索引恢复照片数据（如果数据库为空）
	if vectorIndex.Size() > 0 && (database == nil || func() bool { c, _ := database.GetPhotoCount(); return c == 0 }()) {
		zapLog.Info("从向量索引恢复照片数据", zap.Int("count", vectorIndex.Size()))
		idx.UpdatePhotos(vectorIndex.GetAllPaths())
	}

	// 从文件恢复标签（兼容旧版本）
	idx.LoadTagsFromFile()
	zapLog.Info("标签文件加载完成")

	// Initialize LLM client for query parsing
	llmClient, err := query.NewLLMClientFromEnv()
	if err != nil {
		zapLog.Warn("LLM client initialization failed, natural language search disabled", zap.Error(err))
	} else {
		zapLog.Info("LLM client initialized", zap.String("model", os.Getenv("OPENAI_MODEL")))
	}

	// Setup router
	r := gin.Default()
	r.Use(middleware.Logger(zapLog))
	r.Use(middleware.Cors())

	// Initialize handlers
	searchHandler := handlers.NewSearchHandlerWithLLM(idx, zapLog, llmClient)
	photosHandler := handlers.NewPhotosHandler(idx, mlClient, zapLog)
	statsHandler := handlers.NewStatsHandler(idx, zapLog)
	chatHandler := handlers.NewChatHandler(idx, zapLog)
	filesHandler := handlers.NewFilesHandler(scanSvc, zapLog)
	systemHandler := handlers.NewSystemHandler(zapLog)

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/search", searchHandler.HandleSearch)
		api.GET("/photos", photosHandler.HandleList)
		api.GET("/photos/:id", photosHandler.HandleGet)
		api.GET("/photos/file", photosHandler.HandleServePhoto)
		api.POST("/photos/classify-all", photosHandler.HandleClassifyAll)
		api.GET("/stats", statsHandler.HandleStats)
		api.POST("/chat", chatHandler.HandleChat)
		api.POST("/files/scan", filesHandler.HandleScan)
		api.GET("/files/scan/status", filesHandler.HandleScanStatus)
		api.GET("/dirs", func(c *gin.Context) {
			dirs := idx.ListDirs()
			c.JSON(http.StatusOK, gin.H{"dirs": dirs})
		})

		// System routes
		api.POST("/system/pick-folder", systemHandler.HandlePickFolder)

		// Face recognition routes
		if database != nil {
			faceHandler := handlers.NewFaceHandler(idx, database, cfg.MLServiceURL, zapLog)
			api.POST("/faces/detect", faceHandler.HandleDetectFaces)
			api.PUT("/faces/:id/label", faceHandler.HandleLabelFace)
			api.GET("/persons", faceHandler.HandleGetPersons)
			api.GET("/search/person", faceHandler.HandleSearchByPerson)
			api.GET("/faces/thumbnail", faceHandler.HandleGetFaceThumbnail)
			api.GET("/faces/stats", faceHandler.HandleGetFaceStats)
			api.GET("/faces/clusters", faceHandler.HandleGetClusters)
			api.POST("/faces/label-cluster", faceHandler.HandleLabelCluster)
		}
	}

	// Static files
	r.Static("/uploads", "./uploads")

	// Start server
	zapLog.Info("Server starting", zap.String("addr", cfg.ServerAddr))
	if err := r.Run(cfg.ServerAddr); err != nil {
		zapLog.Fatal("Server failed to start", zap.Error(err))
	}
}
