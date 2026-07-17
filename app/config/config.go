package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort            string
	AppEnv             string
	DBPath             string
	SessionSecret      string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	FrontendURL        string
	// CORS
	AllowedOrigins []string
	// Email configuration
	SMTPHost  string
	SMTPPort  int
	SMTPUser  string
	SMTPPass  string
	FromEmail string
	FromName  string
	// Session
	SessionTTL time.Duration
}

var AppConfig *Config

func Load() *Config {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("no .env file found, using environment variables")
	}

	AppConfig = &Config{
		AppPort:            getEnv("APP_PORT", "8080"),
		AppEnv:             getEnv("APP_ENV", "development"),
		DBPath:             getEnv("DB_PATH", "./data/app.db"),
		SessionSecret:      getEnv("SESSION_SECRET", "change-this-in-production"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		AllowedOrigins:     parseAllowedOrigins(),
		// Email configuration
		SMTPHost:  getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:  getEnvAsInt("SMTP_PORT", 587),
		SMTPUser:  getEnv("SMTP_USER", ""),
		SMTPPass:  getEnv("SMTP_PASS", ""),
		FromEmail: getEnv("FROM_EMAIL", "noreply@example.com"),
		FromName:  getEnv("FROM_NAME", "Laju"),
		// Session
		SessionTTL: getSessionTTL(),
	}

	return AppConfig
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return defaultValue
	}
	return result
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf("file:%s?_foreign_keys=on", c.DBPath)
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// parseAllowedOrigins parses ALLOWED_ORIGINS env var (comma-separated).
// Defaults to http://localhost:5173 in dev, empty (strict) in prod.
func parseAllowedOrigins() []string {
	val := os.Getenv("ALLOWED_ORIGINS")
	if val == "" {
		return []string{"http://localhost:5173"}
	}
	var origins []string
	for _, o := range strings.Split(val, ",") {
		trimmed := strings.TrimSpace(o)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}

func getSessionTTL() time.Duration {
	val := getEnv("SESSION_TTL", "24h")
	d, err := time.ParseDuration(val)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}
