package config

import (
	"os"
	"path/filepath"

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
