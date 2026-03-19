package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	APIKey        string
	EncryptionKey []byte
	WhatsApp      WhatsAppConfig
	LogLevel      string
}

type ServerConfig struct {
	Port int
	Host string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type WhatsAppConfig struct {
	DeviceStorePath string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port: getEnvInt("ONDAPILE_PORT", 8080),
			Host: getEnv("ONDAPILE_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "ondapile"),
			Password: getEnv("DB_PASSWORD", "ondapile"),
			Name:     getEnv("DB_NAME", "ondapile"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		APIKey: getEnv("ONDAPILE_API_KEY", ""),
		WhatsApp: WhatsAppConfig{
			DeviceStorePath: getEnv("WA_DEVICE_STORE_PATH", "./devices"),
		},
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	// Derive encryption key from passphrase (or generate a default for dev)
	encKeyPassphrase := getEnv("ONDAPILE_ENCRYPTION_KEY", cfg.APIKey)
	cfg.EncryptionKey = DeriveKey(encKeyPassphrase)

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ONDAPILE_API_KEY environment variable is required")
	}

	return cfg, nil
}

func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
