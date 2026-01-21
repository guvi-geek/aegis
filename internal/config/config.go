package config

import (
	"fmt"
	"time"

	"github.com/RishiKendai/aegis/internal/configs/env"
)

// Config holds all configuration for the application
type Config struct {
	// MongoDB
	MongoURI    string
	MongoDBName string

	// Redis
	RedisHost               string
	RedisPassword           string
	RedisStreamKey          string
	RedisConsumerGroup      string
	RedisDeadLetterKey      string
	StreamRetentionDuration time.Duration

	// Astra Service
	AstraBaseURL string
	AstraAPIKey  string

	// JWT
	JWTSecret string
	JWTIssuer string

	// Rate Limiting
	RateLimitRPS float64

	// Concurrency
	MaxConcurrentCompute int

	// Computation
	ComputationTimeout time.Duration
	BatchSize          int

	// Test Risk Thresholds
	TestRiskSafe     float64
	TestRiskModerate float64
	TestRiskHigh     float64
	TestRiskCritical float64

	// Logging
	LogLevel string

	// Server
	ServerPort string
}

func Load() (*Config, error) {
	cfg := &Config{}

	// MongoDB

	cfg.MongoURI = env.GetEnv("MONGO_URI", "")
	cfg.MongoDBName = env.GetEnv("MONGO_DB_NAME", "")

	// Redis
	cfg.RedisHost = env.GetEnv("REDIS_HOST", "localhost:6379")
	cfg.RedisPassword = env.GetEnv("REDIS_PASSWORD", "")
	cfg.RedisStreamKey = env.GetEnv("REDIS_STREAM_KEY", "plagiarism:stream")
	cfg.RedisConsumerGroup = env.GetEnv("REDIS_CONSUMER_GROUP", "plagiarism:group")
	cfg.RedisDeadLetterKey = env.GetEnv("REDIS_DEAD_LETTER_KEY", "plagiarism:dlq")
	retentionHours := env.GetEnvInt("STREAM_RETENTION_DURATION", 24)
	cfg.StreamRetentionDuration = time.Duration(retentionHours) * time.Hour

	// Astra Service
	cfg.AstraBaseURL = env.GetEnv("ASTRA_BASE_URL", "")
	cfg.AstraAPIKey = env.GetEnv("ASTRA_API_KEY", "")

	// JWT
	cfg.JWTSecret = env.GetEnv("JWT_SECRET", "")
	cfg.JWTIssuer = env.GetEnv("JWT_ISSUER", "aegis")

	// Rate Limiting
	cfg.RateLimitRPS = env.GetEnvFloat("RATE_LIMIT_RPS", 10.0)

	// Concurrency
	cfg.MaxConcurrentCompute = env.GetEnvInt("MAX_CONCURRENT_COMPUTE", 5)

	// Computation
	timeoutMinutes := env.GetEnvInt("COMPUTATION_TIMEOUT_MINUTES", 30)
	cfg.ComputationTimeout = time.Duration(timeoutMinutes) * time.Minute
	cfg.BatchSize = env.GetEnvInt("BATCH_SIZE", 100)

	// Test Risk Thresholds
	cfg.TestRiskSafe = env.GetEnvFloat("TEST_RISK_SAFE", 0.0)
	cfg.TestRiskModerate = env.GetEnvFloat("TEST_RISK_MODERATE", 0.3)
	cfg.TestRiskHigh = env.GetEnvFloat("TEST_RISK_HIGH", 0.7)
	cfg.TestRiskCritical = env.GetEnvFloat("TEST_RISK_CRITICAL", 0.9)

	// Logging
	cfg.LogLevel = env.GetEnv("LOG_LEVEL", "info")

	// Server
	cfg.ServerPort = env.GetEnv("SERVER_PORT", "8080")

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.MongoURI == "" {
		return fmt.Errorf("MONGODB_URI is required")
	}
	if c.MongoDBName == "" {
		return fmt.Errorf("MONGODB_DB_NAME is required")
	}
	if c.RedisHost == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if c.AstraBaseURL == "" {
		return fmt.Errorf("ASTRA_BASE_URL is required")
	}
	if c.AstraAPIKey == "" {
		return fmt.Errorf("ASTRA_API_KEY is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.MaxConcurrentCompute <= 0 {
		return fmt.Errorf("MAX_CONCURRENT_COMPUTE must be greater than 0")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("BATCH_SIZE must be greater than 0")
	}
	if c.StreamRetentionDuration <= 0 {
		return fmt.Errorf("STREAM_RETENTION_HOURS must be greater than 0")
	}
	return nil
}
