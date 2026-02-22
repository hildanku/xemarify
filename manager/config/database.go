package config

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
