package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port         string
	JWTSecret    string
	JWTExpiry    int // in hours
	LogLevel     string
	MaxMessageLength int
}

func Load() Config {
	port := getEnv("PORT", "8081")
	secret := getEnv("JWT_SECRET", "dev-super-secret-change-me")
	jwtExpiry := getEnvAsInt("JWT_EXPIRY", 24)
	logLevel := getEnv("LOG_LEVEL", "info")
	maxMsgLen := getEnvAsInt("MAX_MESSAGE_LENGTH", 1000)
	
	return Config{
		Port:         port,
		JWTSecret:    secret,
		JWTExpiry:    jwtExpiry,
		LogLevel:     logLevel,
		MaxMessageLength: maxMsgLen,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
