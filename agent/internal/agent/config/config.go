package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Endpoint string `yaml:"endpoint"`
		Insecure bool   `yaml:"insecure"`
	} `yaml:"server"`
	EnrollmentToken string `yaml:"enrollment_token"`
	DiskBuffer      struct {
		Path     string `yaml:"path"`
		MaxBytes int64  `yaml:"max_bytes"`
	} `yaml:"disk_buffer"`
	Agent struct {
		ID          string `yaml:"id"`
		AgentSecret string `yaml:"agent_secret"`
		Name        string `yaml:"name"`
		Hostname    string `yaml:"hostname"`
		IPAddress   string `yaml:"ip_address"`
	} `yaml:"agent"`
	FileLog struct {
		Enabled      bool          `yaml:"enabled"`
		Paths        []string      `yaml:"paths"`
		PollInterval time.Duration `yaml:"poll_interval"`
	} `yaml:"filelog"`
	Inventory struct {
		Enabled  bool          `yaml:"enabled"`
		Interval time.Duration `yaml:"interval"`
	} `yaml:"inventory"`
	Syslog struct {
		Listen string `yaml:"listen"`
	} `yaml:"syslog"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Syslog.Listen == "" {
		cfg.Syslog.Listen = ":5514"
	}

	if cfg.FileLog.PollInterval <= 0 {
		cfg.FileLog.PollInterval = 1 * time.Second
	}

	if cfg.Inventory.Interval <= 0 {
		cfg.Inventory.Interval = 5 * time.Minute
	}

	if cfg.DiskBuffer.Path == "" {
		cfg.DiskBuffer.Path = "/var/lib/xemarify-agent/spool/events.log"
	}

	if cfg.DiskBuffer.MaxBytes <= 0 {
		cfg.DiskBuffer.MaxBytes = 500 * 1024 * 1024
	}

	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
