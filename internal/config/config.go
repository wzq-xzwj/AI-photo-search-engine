package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// Config 应用配置
type Config struct {
	ServerAddr      string   `json:"server_addr"`
	ScanDirs        []string `json:"scan_dirs"`
	IndexDir        string   `json:"index_dir"`
	ExtractEXIF     bool     `json:"extract_exif"`
	MLServiceURL    string   `json:"ml_service_url"`
	VectorIndexPath string   `json:"vector_index_path"`
	DBPath          string   `json:"db_path"`
	LabelSpacePath  string   `json:"label_space_path"`
	LogLevel        string   `json:"log_level"`
}

var (
	instance *Config
	once     sync.Once
)

// Get 获取单例配置（线程安全）
func Get() *Config {
	once.Do(func() {
		instance = load()
	})
	return instance
}

// Reset 重置单例（主要用于测试）
func Reset() {
	once = sync.Once{}
	instance = nil
}

// load 加载配置（合并文件 + 环境变量）
func load() *Config {
	cfg := &Config{
		ServerAddr:      ":8080",
		ScanDirs:        []string{},
		IndexDir:        "data/index",
		ExtractEXIF:     true,
		MLServiceURL:    "http://127.0.0.1:8000",
		VectorIndexPath: "data/vector_index.json",
		DBPath:          "data/photos.db",
		LabelSpacePath:  "config/label_space_v1.json",
		LogLevel:        "info",
	}

	// 从配置文件加载
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.json"
	}

	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, cfg) // 忽略解析错误，使用默认值
	}

	// 环境变量覆盖（优先级最高）
	if v := os.Getenv("SERVER_ADDR"); v != "" {
		cfg.ServerAddr = v
	}
	if v := os.Getenv("ML_SERVICE_URL"); v != "" {
		cfg.MLServiceURL = v
	}
	if v := os.Getenv("VECTOR_INDEX_PATH"); v != "" {
		cfg.VectorIndexPath = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("LABEL_SPACE_PATH"); v != "" {
		cfg.LabelSpacePath = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("INDEX_DIR"); v != "" {
		cfg.IndexDir = v
	}
	if v, err := strconv.ParseBool(os.Getenv("EXTRACT_EXIF")); err == nil {
		cfg.ExtractEXIF = v
	}

	// 确保目录存在
	os.MkdirAll(cfg.IndexDir, 0755)
	os.MkdirAll(filepath.Dir(cfg.DBPath), 0755)
	os.MkdirAll(filepath.Dir(cfg.VectorIndexPath), 0755)

	// 默认扫描目录
	if len(cfg.ScanDirs) == 0 {
		home, _ := os.UserHomeDir()
		cfg.ScanDirs = []string{filepath.Join(home, "Pictures")}
	}

	return cfg
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
