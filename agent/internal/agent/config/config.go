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
	Agent struct {
		ID       string `yaml:"id"`
		Key      string `yaml:"key"`
		AgentKey string `yaml:"agent_key"`
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
