package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// 从数据库加载照片数据（优先），并恢复 embedding
	dbLoaded := false
	var dbPhotos []indexer.Photo
	if database != nil {
		photos, err := database.LoadPhotos()
		if err != nil {
			zapLog.Error("从数据库加载照片失败", zap.Error(err))
		} else {
			zapLog.Info("从数据库恢复照片数据", zap.Int("count", len(photos)))
			if len(photos) > 0 {
				idx.UpdatePhotosMeta(photos)
				dbLoaded = true
				dbPhotos = photos
			}
		}
	}

	// 如果 Milvus 可用，将 SQLite 中的 embedding 批量导入 Milvus
	if dbLoaded && vectorIndex != nil && vectorIndex.Size() == 0 && len(dbPhotos) > 0 {
		var batchItems []internal.ImageEmbedding
		for _, p := range dbPhotos {
			if len(p.Embedding) > 0 {
				batchItems = append(batchItems, internal.ImageEmbedding{
					Path:      p.Path,
					Embedding: p.Embedding,
				})
			}
		}
		if len(batchItems) > 0 {
			vectorIndex.BatchAdd(batchItems)
			zapLog.Info("从 SQLite 恢复 embedding 到 Milvus", zap.Int("count", len(batchItems)))
		}
	}

	// 从向量索引恢复照片数据（如果数据库为空）
	if vectorIndex.Size() > 0 && (database == nil || func() bool { c, _ := database.GetPhotoCount(); return c == 0 }()) {
		zapLog.Info("从向量索引恢复照片数据", zap.Int("count", vectorIndex.Size()))
		idx.UpdatePhotos(vectorIndex.GetAllPaths())
	}

	// 仅在数据库未加载时从文件恢复标签（避免覆盖数据库中的丰富标签）
	if !dbLoaded {
		idx.LoadTagsFromFile()
		zapLog.Info("标签文件加载完成")
	} else {
		zapLog.Info("跳过标签文件加载（数据库已有完整标签）")
	}

	// 同步人物数据（persons 表）
	if database != nil {
		personCount, err := database.SyncPersonsFromFaces()
		if err != nil {
			zapLog.Warn("同步人物数据失败", zap.Error(err))
		} else {
			zapLog.Info("人物数据同步完成", zap.Int("count", personCount))
		}
	}

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
	photosHandler := handlers.NewPhotosHandler(idx, mlClient, database, zapLog)
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

	// Create http.Server
	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		zapLog.Info("Server starting", zap.String("addr", cfg.ServerAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLog.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zapLog.Info("Server shutting down gracefully...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zapLog.Error("Server forced to shutdown", zap.Error(err))
	}

	// Close vector index
	if err := vectorIndex.Close(); err != nil {
		zapLog.Error("Failed to close vector index", zap.Error(err))
	}

	zapLog.Info("Server exited")
}
