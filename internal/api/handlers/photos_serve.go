package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// 允许的照片根目录列表，防止路径遍历
var allowedPhotoRoots = []string{
	"/Volumes/",
	"/Users/",
	"/tmp/",
	"/data/",
	"/photos/",
}

func isPathAllowed(p string) bool {
	clean := filepath.Clean(p)
	// 禁止绝对路径跳出允许范围
	if strings.Contains(clean, "..") {
		return false
	}
	for _, root := range allowedPhotoRoots {
		if strings.HasPrefix(clean, root) {
			return true
		}
	}
	return false
}

// HandleServePhoto 代理返回照片文件
func (h *PhotosHandler) HandleServePhoto(c *gin.Context) {
	// 获取照片路径
	photoPath := c.Query("path")
	if photoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter is required"})
		return
	}

	// 安全检查：路径白名单
	if !isPathAllowed(photoPath) {
		c.JSON(http.StatusForbidden, gin.H{"error": "path not allowed"})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(photoPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(photoPath))

	// 设置 Content-Type
	contentType := "image/jpeg"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".bmp":
		contentType = "image/bmp"
	case ".tiff", ".tif":
		contentType = "image/tiff"
	}

	// 返回文件
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(photoPath)
}
