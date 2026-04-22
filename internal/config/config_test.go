package config

import (
	"os"
	"testing"
)

func TestGetSingleton(t *testing.T) {
	Reset()
	cfg1 := Get()
	cfg2 := Get()
	if cfg1 != cfg2 {
		t.Error("Get() should return the same instance")
	}
}

func TestDefaultValues(t *testing.T) {
	Reset()
	cfg := Get()
	if cfg.ServerAddr != ":8080" {
		t.Errorf("expected ServerAddr ':8080', got '%s'", cfg.ServerAddr)
	}
	if cfg.MLServiceURL != "http://127.0.0.1:8000" {
		t.Errorf("expected MLServiceURL 'http://127.0.0.1:8000', got '%s'", cfg.MLServiceURL)
	}
	if cfg.DBPath != "data/photos.db" {
		t.Errorf("expected DBPath 'data/photos.db', got '%s'", cfg.DBPath)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("SERVER_ADDR", ":9090")
	os.Setenv("ML_SERVICE_URL", "http://localhost:9999")
	defer os.Unsetenv("SERVER_ADDR")
	defer os.Unsetenv("ML_SERVICE_URL")

	Reset()
	cfg := Get()
	if cfg.ServerAddr != ":9090" {
		t.Errorf("expected ServerAddr ':9090', got '%s'", cfg.ServerAddr)
	}
	if cfg.MLServiceURL != "http://localhost:9999" {
		t.Errorf("expected MLServiceURL 'http://localhost:9999', got '%s'", cfg.MLServiceURL)
	}
}
