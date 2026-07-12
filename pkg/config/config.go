package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AppConfig holds all application configuration
type AppConfig struct {
	App struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Env     string `json:"env"` // dev, prod
	} `json:"app"`

	Watcher struct {
		DebounceDelay   time.Duration `json:"debounce_delay_ms"`
		MaxQueueSize    int           `json:"max_queue_size"`
		IncludeExts     []string      `json:"include_exts"`
		Recursive       bool          `json:"recursive"`
		PollingInterval time.Duration `json:"polling_interval_ms"`
	} `json:"watcher"`

	Log struct {
		Level      string `json:"level"` // debug, info, error
		FilePath   string `json:"file_path"`
		MaxSizeMB  int    `json:"max_size_mb"`
		MaxBackups int    `json:"max_backups"`
	} `json:"log"`

	RateLimit struct {
		RequestsPerSec float64 `json:"requests_per_sec"`
		Burst          int     `json:"burst"`
	} `json:"rate_limit"`
}

var (
	instance *AppConfig
	once     sync.Once
)

// Get returns the singleton config instance
func Get() *AppConfig {
	once.Do(func() {
		instance = loadConfig()
	})
	return instance
}

// loadConfig loads configuration with defaults and overrides from file
func loadConfig() *AppConfig {
	// Defaults
	cfg := &AppConfig{}
	cfg.App.Name = "PH Sovereign UI"
	cfg.App.Version = "1.0.0"
	cfg.App.Env = "prod"

	cfg.Watcher.DebounceDelay = 300 * time.Millisecond
	cfg.Watcher.MaxQueueSize = 1000
	cfg.Watcher.IncludeExts = []string{".xlsx", ".pdf", ".docx", ".xml", ".csv", ".json"}
	cfg.Watcher.Recursive = true
	cfg.Watcher.PollingInterval = 0

	cfg.Log.Level = "info"
	cfg.Log.FilePath = "sovereign_ui.log"
	cfg.Log.MaxSizeMB = 10
	cfg.Log.MaxBackups = 3

	cfg.RateLimit.RequestsPerSec = 10.0
	cfg.RateLimit.Burst = 20

	// Try to load from config.json
	execPath, err := os.Executable()
	if err == nil {
		configPath := filepath.Join(filepath.Dir(execPath), "config.json")
		if file, err := os.ReadFile(configPath); err == nil {
			// We use a temporary struct to handle time.Duration unmarshalling if needed
			// For simplicity, we'll just unmarshal what matches
			if err := json.Unmarshal(file, cfg); err != nil {
				// Don't fail hard, but log the error so user knows their config is broken
				// Using fmt.Printf since logger may not be initialized yet
				fmt.Printf("Warning: failed to parse config file %s: %v (using defaults)\n", configPath, err)
			}

			// Fix up duration fields if they came in as raw numbers (e.g. 300 -> 300ns)
			// In a robust system, we'd implement custom UnmarshalJSON
			if cfg.Watcher.DebounceDelay < time.Millisecond {
				cfg.Watcher.DebounceDelay *= time.Millisecond
			}
			if cfg.Watcher.PollingInterval > 0 && cfg.Watcher.PollingInterval < time.Millisecond {
				cfg.Watcher.PollingInterval *= time.Millisecond
			}
		}
	}

	return cfg
}
