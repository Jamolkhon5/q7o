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
	Push     PushConfig
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
	PublicHost string
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

type PushConfig struct {
	FirebaseCredentialsPath string
	FirebaseProjectID       string
	APNsAuthToken           string
	APNsVoIPAuthToken       string
	APNsBundleID            string
	APNsVoIPBundleID        string
	APNsSandbox             bool
}

func Load() *Config {
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
			PublicHost: getEnv("LIVEKIT_PUBLIC_HOST", ""),
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
		Push: PushConfig{
			FirebaseCredentialsPath: getEnv("FIREBASE_CREDENTIALS_PATH", ""),
			FirebaseProjectID:       getEnv("FIREBASE_PROJECT_ID", ""),
			APNsAuthToken:           getEnv("APNS_AUTH_TOKEN", ""),
			APNsVoIPAuthToken:       getEnv("APNS_VOIP_AUTH_TOKEN", ""),
			APNsBundleID:            getEnv("APNS_BUNDLE_ID", "com.q7o.app"),
			APNsVoIPBundleID:        getEnv("APNS_VOIP_BUNDLE_ID", "com.q7o.app.voip"),
			APNsSandbox:             getEnv("APNS_SANDBOX", "true") == "true",
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
