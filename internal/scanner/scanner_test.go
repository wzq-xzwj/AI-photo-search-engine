package scanner

import (
	"context"
	"testing"
	"time"

	"photo-search-engine/internal"
	"photo-search-engine/internal/indexer"
)

func TestNew(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("expected non-nil Scanner")
	}
}

func TestNewWithML(t *testing.T) {
	idx := indexer.New()
	mlClient := internal.NewMLClient("")
	vectorIndex := internal.NewVectorIndex("")

	s := NewWithML(idx, mlClient, vectorIndex)
	if s == nil {
		t.Fatal("expected non-nil Scanner")
	}
}

func TestStartScan(t *testing.T) {
	s := New()
	tmpDir := t.TempDir()

	taskID := s.StartScan(tmpDir)
	if taskID == "" {
		t.Fatal("expected non-empty task ID")
	}

	// 等待扫描完成
	time.Sleep(100 * time.Millisecond)

	task := s.GetTask(taskID)
	if task == nil {
		t.Fatal("expected task to exist")
	}

	// 应该完成或失败（空目录）
	if task.Status != "completed" && task.Status != "failed" {
		t.Errorf("expected completed or failed, got %s", task.Status)
	}
}

func TestStartScanWithContext(t *testing.T) {
	s := New()
	tmpDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	taskID := s.StartScanWithContext(ctx, tmpDir)

	// 立即取消
	cancel()
	time.Sleep(200 * time.Millisecond)

	task := s.GetTask(taskID)
	if task == nil {
		t.Fatal("expected task to exist")
	}

	// 空目录扫描很快，可能已完成；如果被取消则显示 cancelled
	if task.Status != "cancelled" && task.Status != "completed" && task.Status != "failed" {
		t.Errorf("expected cancelled/completed/failed, got %s", task.Status)
	}
}

func TestIsScanning(t *testing.T) {
	s := New()
	if s.IsScanning() {
		t.Error("expected no scanning tasks initially")
	}
}

func TestGetProgressChannel(t *testing.T) {
	s := New()
	tmpDir := t.TempDir()

	taskID := s.StartScan(tmpDir)
	ch := s.GetProgressChannel(taskID, 50*time.Millisecond)

	var lastProgress Progress
	for p := range ch {
		lastProgress = p
	}

	// 应该收到至少一个进度更新
	if lastProgress.Message == "" {
		t.Error("expected non-empty message in progress")
	}
}

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"balloon-test.jpg", true},
		{"test_display_1.png", true},
		{"icon-app.ico", true},
		{"logo.png", true},
		{"favicon.ico", true},
		{"normal.jpg", false},
		{"photo.png", false},
	}

	for _, tt := range tests {
		if got := shouldSkipFile(tt.name); got != tt.expected {
			t.Errorf("shouldSkipFile(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

func TestExtractDirTags(t *testing.T) {
	tests := []struct {
		photoPath string
		baseDir   string
		expected  string
	}{
		{"/photos/vacation/img1.jpg", "/photos", "vacation"},
		{"/photos/family/img2.jpg", "/photos", "family"},
	}

	for _, tt := range tests {
		tags := extractDirTags(tt.photoPath, tt.baseDir)
		if len(tags) != 1 || tags[0] != tt.expected {
			t.Errorf("extractDirTags(%q, %q) = %v, want [%s]", tt.photoPath, tt.baseDir, tags, tt.expected)
		}
	}
}
