package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	ID          string `yaml:"id" json:"id"`
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
}

type ServerConfig struct {
	HTTPPort int    `yaml:"http_port" json:"http_port"`
	WSSPort  int    `yaml:"wss_port" json:"wss_port"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
}

type LogConfig struct {
	BaseDir       string `yaml:"base_dir" json:"base_dir"`
	MaxRetainDays int    `yaml:"max_retain_days" json:"max_retain_days"`
	CleanInterval int    `yaml:"clean_interval_minutes" json:"clean_interval_minutes"`
}

type AuthConfig struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
	JWTSecret string `yaml:"jwt_secret" json:"jwt_secret"`
}

type Config struct {
	Server ServerConfig `yaml:"server" json:"server"`
	Log    LogConfig    `yaml:"log" json:"log"`
	Auth   AuthConfig   `yaml:"auth" json:"auth"`
	Apps   []AppConfig  `yaml:"apps" json:"apps"`
}

var (
	instance *Config
	once     sync.Once
	mu       sync.RWMutex
)

func Load(path string) (*Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	mu.Lock()
	instance = &cfg
	mu.Unlock()

	return &cfg, nil
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	return instance
}

func (c *Config) validate() error {
	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d", c.Server.HTTPPort)
	}
	if c.Server.WSSPort <= 0 || c.Server.WSSPort > 65535 {
		return fmt.Errorf("invalid wss_port: %d", c.Server.WSSPort)
	}
	if c.Log.BaseDir == "" {
		return fmt.Errorf("log base_dir is required")
	}
	if c.Log.MaxRetainDays <= 0 {
		return fmt.Errorf("max_retain_days must be positive")
	}
	if c.Log.CleanInterval <= 0 {
		c.Log.CleanInterval = 60
	}
	if len(c.Apps) == 0 {
		return fmt.Errorf("at least one app must be configured")
	}
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return fmt.Errorf("auth username and password are required")
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "loghub-default-secret-change-me"
	}

	seen := make(map[string]bool)
	for _, app := range c.Apps {
		if app.ID == "" {
			return fmt.Errorf("app id is required")
		}
		if seen[app.ID] {
			return fmt.Errorf("duplicate app id: %s", app.ID)
		}
		seen[app.ID] = true
	}

	return nil
}

func (c *Config) IsAppAllowed(appID string) bool {
	for _, app := range c.Apps {
		if app.ID == appID {
			return true
		}
	}
	return false
}

func (c *Config) GetApp(appID string) *AppConfig {
	for _, app := range c.Apps {
		if app.ID == appID {
			return &app
		}
	}
	return nil
}

func (c *Config) GetAllowedAppIDs() []string {
	ids := make([]string, len(c.Apps))
	for i, app := range c.Apps {
		ids[i] = app.ID
	}
	return ids
}
