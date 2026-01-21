package api

import (
	"github.com/RishiKendai/aegis/internal/config"
	"github.com/RishiKendai/aegis/internal/plagiarism"
	"github.com/RishiKendai/aegis/internal/repository"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	cfg *config.Config,
	artifactsRepo *repository.ArtifactsRepository,
	resultsRepo *repository.ResultsRepository,
	workerPool *plagiarism.WorkerPool,
) *gin.Engine {
	router := gin.Default()

	// Create handler
	handler := NewHandler(cfg, artifactsRepo, resultsRepo, workerPool)

	// Create rate limiter
	rateLimiter := NewRateLimiter(cfg.RateLimitRPS, int(cfg.RateLimitRPS*2))

	// Middleware
	router.Use(ErrorHandlerMiddleware())

	// Health endpoint (no auth)
	router.GET("/health", handler.Health)

	// API routes (with auth and rate limiting)
	api := router.Group("/api/v1")
	api.Use(JWTAuthMiddleware(cfg.JWTSecret))
	api.Use(RateLimitMiddleware(rateLimiter))
	{
		api.POST("/compute", handler.Compute)
	}

	return router
}
