package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

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

// RAW/TIFF 格式列表
var rawFormats = map[string]bool{
	".dng": true, ".cr2": true, ".cr3": true, ".nef": true, ".arw": true,
	".raf": true, ".orf": true, ".rw2": true, ".pef": true, ".srw": true,
	".tiff": true, ".tif": true,
}

// 缩略图缓存目录
var thumbnailCacheDir = "./data/thumbnails"
var thumbnailOnce sync.Once

func ensureThumbnailDir() {
	thumbnailOnce.Do(func() {
		os.MkdirAll(thumbnailCacheDir, 0755)
	})
}

// generateThumbnail 使用 sips 生成 JPEG 缩略图
func generateThumbnail(photoPath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(photoPath))
	if !rawFormats[ext] {
		return "", fmt.Errorf("not a raw format")
	}

	ensureThumbnailDir()

	// 用文件路径的 hash 作为缓存文件名
	cacheName := strings.ReplaceAll(strings.TrimPrefix(photoPath, "/"), "/", "_")
	cacheName = strings.TrimSuffix(cacheName, ext) + ".jpg"
	cachePath := filepath.Join(thumbnailCacheDir, cacheName)

	// 如果缓存已存在，直接返回
	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	// 确保缓存目录存在
	os.MkdirAll(filepath.Dir(cachePath), 0755)

	// 使用 sips 转换（macOS 内置）
	cmd := exec.Command("sips",
		"-s", "format", "jpeg",
		"-s", "formatOptions", "75",
		"-Z", "1200", // 最大边 1200px
		photoPath,
		"--out", cachePath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("sips failed: %v, output: %s", err, string(output))
	}

	return cachePath, nil
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

	// RAW/TIFF 格式：生成 JPEG 缩略图
	if rawFormats[ext] {
		thumbnailPath, err := generateThumbnail(photoPath)
		if err == nil {
			c.Header("Content-Type", "image/jpeg")
			c.Header("Cache-Control", "public, max-age=86400")
			c.File(thumbnailPath)
			return
		}
		// 降级：返回原文件
	}

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
