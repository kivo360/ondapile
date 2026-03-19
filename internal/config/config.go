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
	Google        GoogleOAuthConfig
	Microsoft     MicrosoftOAuthConfig
	LinkedIn      LinkedInOAuthConfig
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

// GoogleOAuthConfig holds Google OAuth configuration.
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// MicrosoftOAuthConfig holds Microsoft OAuth configuration.
type MicrosoftOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	TenantID     string
}

// LinkedInOAuthConfig holds LinkedIn OAuth configuration.
type LinkedInOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
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
		Google: GoogleOAuthConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		},
		Microsoft: MicrosoftOAuthConfig{
			ClientID:     getEnv("MICROSOFT_CLIENT_ID", ""),
			ClientSecret: getEnv("MICROSOFT_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("MICROSOFT_REDIRECT_URL", ""),
			TenantID:     getEnv("MICROSOFT_TENANT_ID", ""),
		},
		LinkedIn: LinkedInOAuthConfig{
			ClientID:     getEnv("LINKEDIN_CLIENT_ID", ""),
			ClientSecret: getEnv("LINKEDIN_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("LINKEDIN_REDIRECT_URL", ""),
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

// DSNURL returns the database connection string in URL format for migrations.
func (d *DatabaseConfig) DSNURL() string {
	return fmt.Sprintf(
		"%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
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
