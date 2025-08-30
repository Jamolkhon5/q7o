package config

import (
	"github.com/joho/godotenv"
	"os"
)

type Config struct {
	AppEnv   string
	AppPort  string
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	LiveKit  LiveKitConfig
	SMTP     SMTPConfig
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret        string
	RefreshSecret string
	ExpireHours   int
	RefreshDays   int
}

type LiveKitConfig struct {
	Host       string
	PublicHost string // ДОБАВЛЕНО
	APIKey     string
	APISecret  string
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func Load() *Config {
	// Try to load .env file, ignore any errors (file might not exist)
	_ = godotenv.Load()

	return &Config{
		AppEnv:  getEnv("APP_ENV", "development"),
		AppPort: getEnv("APP_PORT", "8080"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "q7o"),
			Password: getEnv("DB_PASSWORD", "q7o_secret"),
			Name:     getEnv("DB_NAME", "q7o_db"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "secret"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "refresh_secret"),
			ExpireHours:   1,
			RefreshDays:   90,
		},
		LiveKit: LiveKitConfig{
			Host:       getEnv("LIVEKIT_HOST", "localhost:7880"),
			PublicHost: getEnv("LIVEKIT_PUBLIC_HOST", ""), // ДОБАВЛЕНО
			APIKey:     getEnv("LIVEKIT_API_KEY", "devkey"),
			APISecret:  getEnv("LIVEKIT_API_SECRET", "secret"),
		},
		SMTP: SMTPConfig{
			Host: getEnv("SMTP_HOST", "smtp.mail.ru"),
			Port: 465,
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("SMTP_FROM", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
