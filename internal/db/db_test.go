package db

import (
	"os"
	"path/filepath"
	"testing"

	"photo-search-engine/internal/indexer"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test db: %v", err)
	}
	return db, func() {
		db.Close()
	}
}

func TestNew(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if db == nil {
		t.Fatal("expected non-nil DB")
	}
}

func TestSaveAndLoadPhoto(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	photo := &indexer.Photo{
		ID:          "photo-1",
		Path:        "/test/photo1.jpg",
		Name:        "photo1.jpg",
		Tags:        []string{"test", "vacation"},
		DateTime:    "2024-01-01 12:00:00",
		Width:       1920,
		Height:      1080,
		CameraMake:  "Canon",
		CameraModel: "EOS R5",
	}

	if err := db.SavePhoto(photo); err != nil {
		t.Fatalf("SavePhoto failed: %v", err)
	}

	photos, err := db.LoadPhotos()
	if err != nil {
		t.Fatalf("LoadPhotos failed: %v", err)
	}
	if len(photos) != 1 {
		t.Fatalf("expected 1 photo, got %d", len(photos))
	}

	loaded := photos[0]
	if loaded.ID != photo.ID {
		t.Errorf("expected ID '%s', got '%s'", photo.ID, loaded.ID)
	}
	if loaded.Path != photo.Path {
		t.Errorf("expected Path '%s', got '%s'", photo.Path, loaded.Path)
	}
	if len(loaded.Tags) != 2 || loaded.Tags[0] != "test" {
		t.Errorf("expected tags [test vacation], got %v", loaded.Tags)
	}
}

func TestSavePhotos(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	photos := []indexer.Photo{
		{ID: "p1", Path: "/a/1.jpg", Name: "1.jpg", Tags: []string{"a"}},
		{ID: "p2", Path: "/b/2.jpg", Name: "2.jpg", Tags: []string{"b"}},
		{ID: "p3", Path: "/c/3.jpg", Name: "3.jpg", Tags: []string{"c"}},
	}

	if err := db.SavePhotos(photos); err != nil {
		t.Fatalf("SavePhotos failed: %v", err)
	}

	count, err := db.GetPhotoCount()
	if err != nil {
		t.Fatalf("GetPhotoCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestUpdatePhotoTags(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	photo := &indexer.Photo{
		ID:   "photo-1",
		Path: "/test/photo1.jpg",
		Name: "photo1.jpg",
		Tags: []string{"old"},
	}
	if err := db.SavePhoto(photo); err != nil {
		t.Fatalf("SavePhoto failed: %v", err)
	}

	if err := db.UpdatePhotoTags("photo-1", []string{"new1", "new2"}); err != nil {
		t.Fatalf("UpdatePhotoTags failed: %v", err)
	}

	tags, err := db.GetTagsByPhotoID("photo-1")
	if err != nil {
		t.Fatalf("GetTagsByPhotoID failed: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestSaveAndLoadVector(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}
	if err := db.SaveVector("/test/img.jpg", embedding); err != nil {
		t.Fatalf("SaveVector failed: %v", err)
	}

	vectors, err := db.LoadVectors()
	if err != nil {
		t.Fatalf("LoadVectors failed: %v", err)
	}
	if len(vectors) != 1 {
		t.Fatalf("expected 1 vector, got %d", len(vectors))
	}

	loaded, ok := vectors["/test/img.jpg"]
	if !ok {
		t.Fatal("expected vector for /test/img.jpg")
	}
	if len(loaded) != len(embedding) {
		t.Errorf("expected %d dims, got %d", len(embedding), len(loaded))
	}
}

func TestSearchHistory(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	if err := db.SaveSearchHistory("query1", 10, map[string]string{"dir": "/test"}); err != nil {
		t.Fatalf("SaveSearchHistory failed: %v", err)
	}

	history, err := db.GetSearchHistory(10)
	if err != nil {
		t.Fatalf("GetSearchHistory failed: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(history))
	}
	if history[0].Query != "query1" {
		t.Errorf("expected query 'query1', got '%s'", history[0].Query)
	}
}

func TestMigrateFromJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建测试 JSON 文件
	photosData := `[{"id":"p1","path":"/a/1.jpg","name":"1.jpg","tags":["tag1"],"date_time":"2024-01-01"}]`
	photosPath := filepath.Join(tmpDir, "photos.json")
	os.WriteFile(photosPath, []byte(photosData), 0644)

	tagsData := `{"/a/1.jpg":["tag1","tag2"]}`
	tagsPath := filepath.Join(tmpDir, "tags.json")
	os.WriteFile(tagsPath, []byte(tagsData), 0644)

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	defer db.Close()

	if err := db.MigrateFromJSON(photosPath, tagsPath); err != nil {
		t.Fatalf("MigrateFromJSON failed: %v", err)
	}

	count, _ := db.GetPhotoCount()
	if count != 1 {
		t.Errorf("expected 1 photo after migration, got %d", count)
	}
}
