package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort   string
	DatabasePath string
	MCPServerURL string
}

func Load() *Config {
	return &Config{
		ServerPort:   getEnv("SERVER_PORT", "8080"),
		DatabasePath: getEnv("DATABASE_PATH", "./data/jobs.db"),
		MCPServerURL: getEnv("MCP_SERVER_URL", "http://localhost:9423"),
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
