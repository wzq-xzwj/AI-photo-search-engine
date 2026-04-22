package handlers

import (
	"context"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SystemHandler struct {
	logger *zap.Logger
}

func NewSystemHandler(logger *zap.Logger) *SystemHandler {
	return &SystemHandler{logger: logger}
}

func (h *SystemHandler) HandlePickFolder(c *gin.Context) {
	var path string
	var err error

	switch runtime.GOOS {
	case "darwin":
		path, err = pickFolderMacOS()
	case "linux":
		path, err = pickFolderLinux()
	case "windows":
		path, err = pickFolderWindows()
	default:
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Unsupported platform"})
		return
	}

	if err != nil {
		h.logger.Warn("Failed to pick folder", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if path == "" {
		c.JSON(http.StatusOK, gin.H{"path": "", "cancelled": true})
		return
	}

	c.JSON(http.StatusOK, gin.H{"path": strings.TrimSpace(path)})
}

func pickFolderMacOS() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-e", `POSIX path of (choose folder with prompt "选择要扫描的照片目录")`)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", nil // timeout = treat as cancelled
		}
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", nil // user cancelled
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func pickFolderLinux() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "zenity", "--file-selection", "--directory")
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", nil
		}
		// zenity returns exit code 1 when cancelled
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

func pickFolderWindows() (string, error) {
	script := `
Add-Type -AssemblyName System.Windows.Forms
$fbd = New-Object System.Windows.Forms.FolderBrowserDialog
$fbd.Description = "选择要扫描的照片目录"
$fbd.ShowNewFolderButton = $false
if ($fbd.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    Write-Output $fbd.SelectedPath
}
`
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", nil
		}
		return "", nil // likely cancelled
	}
	return strings.TrimSpace(string(out)), nil
}
