package env

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or failed to load, using system environment variables")
		return nil // Don't return error - system env vars will be used via os.Getenv()
	}

	return nil
}

func GetEnv(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func GetEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
