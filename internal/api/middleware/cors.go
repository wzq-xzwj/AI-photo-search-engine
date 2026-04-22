package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// getAllowedOrigins 获取允许的 CORS 来源
// 优先从环境变量 CORS_ORIGINS 读取（逗号分隔），否则使用默认值
func getAllowedOrigins() []string {
	if env := os.Getenv("CORS_ORIGINS"); env != "" {
		origins := strings.Split(env, ",")
		for i := range origins {
			origins[i] = strings.TrimSpace(origins[i])
		}
		return origins
	}
	return []string{
		"http://localhost:5173",  // Vite 开发服务器
		"http://localhost:3000",  // 生产构建
		"http://127.0.0.1:5173",
		"http://127.0.0.1:3000",
	}
}

func Cors() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     getAllowedOrigins(),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
