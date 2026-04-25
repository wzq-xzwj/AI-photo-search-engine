package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"photo-search-engine/internal/scanner"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// allowedBaseDirs 定义允许扫描的白名单目录
var allowedBaseDirs = []string{}

func init() {
	// 初始化白名单目录
	home, _ := os.UserHomeDir()
	allowedBaseDirs = []string{
		filepath.Join(home, "Pictures"),
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Documents"),
		home, // 允许 home 目录本身（但会受后续校验限制）
	}
}

type FilesHandler struct {
	scanner *scanner.Scanner
	logger  *zap.Logger
}

func NewFilesHandler(scannerSvc *scanner.Scanner, logger *zap.Logger) *FilesHandler {
	return &FilesHandler{scanner: scannerSvc, logger: logger}
}

func expandPath(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return os.ExpandEnv(path)
}

// sanitizeScanPath 验证并清理用户输入的扫描路径
// 1. 解析为绝对路径
// 2. 清理路径（消除 .. 等）
// 3. 验证路径在白名单目录内
// 4. 验证路径存在且是目录
func sanitizeScanPath(input string) (string, error) {
	// 展开 ~ 和环境变量
	expanded := expandPath(input)

	// 转换为绝对路径
	absPath, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// 清理路径，消除 .. 等穿越组件
	cleanPath := filepath.Clean(absPath)

	// 解析符号链接（防止通过 symlink 绕过限制）
	realPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		// 如果路径不存在，至少用清理后的路径继续校验
		realPath = cleanPath
	}

	// 验证路径在白名单内
	home, _ := os.UserHomeDir()
	inWhitelist := false
	for _, base := range allowedBaseDirs {
		// 确保 base 也以分隔符结尾，防止前缀匹配漏洞
		baseClean := filepath.Clean(base)
		if realPath == baseClean || strings.HasPrefix(realPath, baseClean+string(filepath.Separator)) {
			inWhitelist = true
			break
		}
	}

	// 额外安全网：禁止访问敏感系统目录
	sensitiveDirs := []string{"/etc", "/usr", "/bin", "/sbin", "/lib", "/lib64", "/proc", "/sys", "/dev", "/var"}
	for _, sensitive := range sensitiveDirs {
		if realPath == sensitive || strings.HasPrefix(realPath, sensitive+string(filepath.Separator)) {
			return "", fmt.Errorf("access to system directory denied: %s", realPath)
		}
	}

	// 允许外置存储 /Volumes（常见照片存储位置）
	inVolumes := strings.HasPrefix(realPath, "/Volumes/")
	if !inWhitelist && !strings.HasPrefix(realPath, home) && !inVolumes {
		return "", fmt.Errorf("path not accessible: %s", realPath)
	}

	// 验证路径存在且是目录
	info, err := os.Stat(realPath)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", realPath)
	}

	return realPath, nil
}

func (h *FilesHandler) HandleScan(c *gin.Context) {
	h.logger.Info("Starting file scan")

	defaultDir := os.Getenv("DEFAULT_SCAN_DIR")
	if defaultDir == "" {
		home, _ := os.UserHomeDir()
		defaultDir = filepath.Join(home, "Pictures")
	}

	rawDir := c.DefaultQuery("dir", defaultDir)
	dir, err := sanitizeScanPath(rawDir)
	if err != nil {
		h.logger.Warn("Invalid scan path rejected", zap.String("input", rawDir), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid directory: %v", err)})
		return
	}

	scanID := h.scanner.StartScan(dir)
	h.logger.Info("File scan started", zap.String("scan_id", scanID), zap.String("directory", dir))

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Scan started",
		"scan_id":    scanID,
		"directory":  dir,
	})
}

func (h *FilesHandler) HandleScanStatus(c *gin.Context) {
	scanID := c.Query("scan_id")
	if scanID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "scan_id is required"})
		return
	}

	task := h.scanner.GetTask(scanID)
	if task == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan task not found"})
		return
	}

	c.JSON(http.StatusOK, task)
}
