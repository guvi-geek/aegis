package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RishiKendai/aegis/internal/api"
	"github.com/RishiKendai/aegis/internal/config"
	"github.com/RishiKendai/aegis/internal/configs/env"
	"github.com/RishiKendai/aegis/internal/infra/mongo"
	redisInfra "github.com/RishiKendai/aegis/internal/infra/redis"
	"github.com/RishiKendai/aegis/internal/logger"
	"github.com/RishiKendai/aegis/internal/metrics"
	"github.com/RishiKendai/aegis/internal/plagiarism"
	"github.com/RishiKendai/aegis/internal/preprocess"
	"github.com/RishiKendai/aegis/internal/repository"
	"github.com/RishiKendai/aegis/internal/stream"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

func main() {
	if err := env.LoadEnv(); err != nil {
		log.Warn().Err(err).Msg("Failed to load .env file, continuing with system environment variables")
	}

	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Invalid configuration: %v", err))
	}

	logger.Init(cfg.LogLevel)
	log.Info().Msg("Starting AEGIS server")
	log.Trace().Msg("🚀 Starting AEGIS server")
	// Initialize Prometheus metrics
	metrics.InitPrometheus()
	log.Info().Msg("Prometheus metrics initialized")

	// Start metrics server in separate goroutine on port 2112
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    ":2112",
		Handler: metricsMux,
	}
	go func() {
		log.Info().Str("port", "2112").Msg("Metrics server started")
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Metrics server failed to start")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect MongoDB
	mongoClient, err := mongo.NewClient(ctx, cfg.MongoURI, cfg.MongoDBName)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create MongoDB client")
	}
	defer mongoClient.Close(ctx)

	// Connect Redis
	redisClient, err := redisInfra.NewClient(ctx, cfg.RedisHost, cfg.RedisPassword, 0)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Redis client")
	}
	defer redisClient.Close()

	// Initialize MongoDB repository
	mongoRepo := repository.NewMongoRepository(mongoClient)

	// Initialize repositories
	artifactsRepo := repository.NewArtifactsRepository(mongoRepo)
	resultsRepo := repository.NewResultsRepository(mongoRepo)

	// Initialize Astra client and preprocessing service
	// astraClient := preprocess.NewAstraClient(cfg.AstraBaseURL, cfg.AstraAPIKey)
	// Use test file instead of real API
	astraClient := preprocess.NewAstraClient(cfg.AstraBaseURL, cfg.AstraAPIKey)

	preprocessSvc := preprocess.NewService(astraClient, artifactsRepo)

	// Initialize retry handler
	retryHandler := stream.NewRetryHandler(redisClient.Client, cfg.RedisDeadLetterKey)

	// Initialize Redis stream consumer
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	consumerName := fmt.Sprintf("consumer-%s-%d-%s", hostname, os.Getpid(), uuid.New().String()[:8])
	consumer := stream.NewConsumer(
		redisClient.Client,
		cfg.RedisStreamKey,
		cfg.RedisConsumerGroup,
		consumerName,
		preprocessSvc,
		retryHandler,
		cfg.StreamRetentionDuration,
	)
	log.Info().Str("consumer_name", consumerName).Msg("Redis stream consumer initialized")

	// Initialize worker pool
	workerPool := plagiarism.NewWorkerPool(ctx)
	defer workerPool.Close()

	router := api.SetupRoutes(cfg, artifactsRepo, resultsRepo, workerPool, redisClient)

	// Start Redis consumer in background
	consumerCtx, consumerCancel := context.WithCancel(ctx)
	go func() {
		defer consumerCancel()
		if err := consumer.Start(consumerCtx); err != nil && err != context.Canceled {
			log.Error().Err(err).Msg("Redis consumer error")
		}
	}()
	log.Info().Msg("Redis consumer started")

	// Start Gin server - Gin handles all HTTP routing, middleware (auth, rate limiter), and request processing
	srv := api.StartServer(router, cfg.ServerPort)

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down gracefully...")

	// Cancel contexts
	// cancel()
	// consumerCancel()

	// Shutdown Gin server gracefully
	if err := api.ShutdownServer(srv, 30*time.Second); err != nil {
		log.Error().Err(err).Msg("Error shutting down Gin server")
	}

	// Shutdown metrics server gracefully
	metricsCtx, metricsCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer metricsCancel()
	if err := metricsServer.Shutdown(metricsCtx); err != nil {
		log.Error().Err(err).Msg("Error shutting down metrics server")
	}

	log.Info().Msg("Shutdown complete")
}
