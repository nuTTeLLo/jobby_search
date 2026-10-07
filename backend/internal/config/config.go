package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort     string
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	DBSSLMode      string
	AllowedOrigins string
	MCPServerURL   string
	JWTSecret      string
	JWTExpiration  time.Duration

	// OAuth sign-in. A provider is offered only when its client ID is set.
	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
	// PublicBaseURL is where the browser reaches this API; the OAuth callback
	// URLs registered with each provider are built from it.
	PublicBaseURL string
	// FrontendURL is where the browser goes once sign-in finishes.
	FrontendURL string
	// AllowedEmails are the only accounts that may sign in.
	AllowedEmails []string
}

func Load() *Config {
	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "jobuser"),
		DBPassword:     getEnv("DB_PASSWORD", "jobpass"),
		DBName:         getEnv("DB_NAME", "jobtracker"),
		DBSSLMode:      getEnv("DB_SSLMODE", "require"),
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		MCPServerURL:   getEnv("MCP_SERVER_URL", "http://localhost:9423"),
		JWTSecret:      getEnv("JWT_SECRET", ""),
		JWTExpiration:  24 * time.Hour,

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GitHubClientID:     getEnv("GITHUB_CLIENT_ID", ""),
		GitHubClientSecret: getEnv("GITHUB_CLIENT_SECRET", ""),
		PublicBaseURL:      getEnv("PUBLIC_BASE_URL", "http://localhost:8080"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:5173"),
		AllowedEmails:      strings.Split(getEnv("ALLOWED_EMAILS", ""), ","),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
