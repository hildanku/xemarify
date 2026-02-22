package config

// define config app

import (
	"fmt"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Database DatabaseConfig
	Server   ServerConfig
}

type ServerConfig struct {
	Host string
	Port int
}

func Load() (*AppConfig, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Warning: .env file not found")
	}

	appcfg := &AppConfig{
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USERNAME"),
			Password: viper.GetString("DB_PASSWORD"),
			Name:     viper.GetString("DB_DATABASE"),
			Schema:   viper.GetString("DB_SCHEMA"),
			SSLMode:  viper.GetString("DB_SSLMODE"),
			MaxConns: int32(viper.GetInt("DB_MAX_CONNS")),
			MinConns: int32(viper.GetInt("DB_MIN_CONNS")),
		},
		Server: ServerConfig{
			Host: viper.GetString("SERVER_HOST"),
			Port: viper.GetInt("PORT"),
		},
	}
	if err := appcfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return appcfg, nil
}

func (c *AppConfig) Validate() error {
	// Database validation
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	return nil
}
