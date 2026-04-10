package config

// define config app

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Database DatabaseConfig
	Server   ServerConfig
	JWT      JWTConfig
	Setup    SetupConfig
	LogLevel string
}

type SetupConfig struct {
	Token string
}

type JWTConfig struct {
	Secret          string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	Schema   string
	SSLMode  string
	MaxConns int32
	MinConns int32
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

	logLevel := viper.GetString("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
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
		JWT: JWTConfig{
			Secret:          viper.GetString("JWT_SECRET"),
			AccessTokenTTL:  parseDurationOrDefault(viper.GetString("JWT_ACCESS_TTL"), 15*time.Minute),
			RefreshTokenTTL: parseDurationOrDefault(viper.GetString("JWT_REFRESH_TTL"), 7*24*time.Hour),
		},
		Setup: SetupConfig{
			Token: viper.GetString("MANAGER_SETUP_TOKEN"),
		},
		LogLevel: logLevel,
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

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required (JWT_SECRET)")
	}

	return nil
}

func parseDurationOrDefault(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
