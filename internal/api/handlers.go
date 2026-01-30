package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/RishiKendai/aegis/internal/config"
	"github.com/RishiKendai/aegis/internal/infra/redis"
	"github.com/RishiKendai/aegis/internal/models"
	"github.com/RishiKendai/aegis/internal/plagiarism"
	"github.com/RishiKendai/aegis/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Handler holds dependencies for handlers
type Handler struct {
	cfg            *config.Config
	artifactsRepo  *repository.ArtifactsRepository
	resultsRepo    *repository.ResultsRepository
	workerPool     *plagiarism.WorkerPool
	redisClient    *redis.Client
	computeSem     chan struct{} // Semaphore for bounded concurrency
	computeTimeout time.Duration
}

// NewHandler creates a new handler
func NewHandler(
	cfg *config.Config,
	artifactsRepo *repository.ArtifactsRepository,
	resultsRepo *repository.ResultsRepository,
	workerPool *plagiarism.WorkerPool,
	redisClient *redis.Client,
) *Handler {
	// Create semaphore for bounded concurrency
	sem := make(chan struct{}, cfg.MaxConcurrentCompute)

	return &Handler{
		cfg:            cfg,
		artifactsRepo:  artifactsRepo,
		resultsRepo:    resultsRepo,
		workerPool:     workerPool,
		redisClient:    redisClient,
		computeSem:     sem,
		computeTimeout: cfg.ComputationTimeout,
	}
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

func (h *Handler) Compute(c *gin.Context) {
	var req models.ComputeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid request body",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	// Input validation
	if err := validateComputePayload(req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_DRIVE_ID",
		})
		return
	}

	// Check if artifacts exist (Edge Case: Missing driveId)
	ctx := c.Request.Context()
	count, err := h.artifactsRepo.CountArtifactsByDriveID(ctx, req.DriveID)
	if err != nil {
		log.Error().Err(err).Str("driveId", req.DriveID).Msg("Failed to check artifacts")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to check artifacts",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if count == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "No artifacts found for driveId",
			Code:  "DRIVE_ID_NOT_FOUND",
		})
		return
	}

	// Check if already completed
	latestReport, err := h.resultsRepo.GetLatestReportByDriveID(ctx, req.DriveID)
	if err != nil {
		log.Error().Err(err).Str("driveId", req.DriveID).Msg("Failed to get latest report")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to check computation status",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	if latestReport != nil && latestReport.Status == "completed" {
		// Update status: Completed
		if err := plagiarism.UpdateStatus(ctx, h.redisClient, req.DriveID, models.StepCompleted); err != nil {
			log.Warn().Err(err).Str("driveId", req.DriveID).Msg("Failed to update completed status")
		}
	}

	// Acquire semaphore (bounded concurrency)
	select {
	case h.computeSem <- struct{}{}:
		// Acquired semaphore
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, ErrorResponse{
			Error: "Request cancelled",
			Code:  "REQUEST_TIMEOUT",
		})
		return
	}

	// Update status: Initiated
	if err := plagiarism.UpdateStatus(ctx, h.redisClient, req.DriveID, models.StepInitiated); err != nil {
		log.Warn().Err(err).Str("driveId", req.DriveID).Msg("Failed to update initiated status")
	}

	// Return 202 Accepted immediately
	c.JSON(http.StatusAccepted, models.ComputeResponse{
		Step:   models.StepInitiated,
		TestID: req.DriveID,
	})

	// Process asynchronously
	go h.processComputation(req.DriveID)
}

// processComputation processes computation asynchronously
func (h *Handler) processComputation(driveID string) {
	defer func() { <-h.computeSem }() // Release semaphore

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.computeTimeout)
	defer cancel()

	// Create pending report
	pendingReport := &models.TestReport{
		DriveID:           driveID,
		Risk:              "",
		Status:            "pending",
		FlaggedQuestions:  []string{},
		FlaggedCandidates: 0,
		TotalAnalyzed:     0,
	}

	if err := h.resultsRepo.InsertTestReport(ctx, pendingReport); err != nil {
		log.Error().Err(err).Str("driveId", driveID).Msg("Failed to create pending report")
	}

	// Process computation
	err := plagiarism.ComputePlagiarism(
		ctx,
		driveID,
		h.artifactsRepo,
		h.resultsRepo,
		h.workerPool,
		h.redisClient,
		h.cfg.BatchSize,
	)

	if err != nil {
		log.Error().Err(err).Str("driveId", driveID).Msg("Computation failed")
		h.createFailedReport(ctx, driveID, err.Error())
		return
	}

	log.Debug().Str("driveId", driveID).Msg("Computation completed successfully")
}

func (h *Handler) createFailedReport(ctx context.Context, driveID, errorMsg string) {
	err := h.resultsRepo.UpdateTestReportByDriveID(ctx, driveID, &models.TestReport{
		DriveID:           driveID,
		Risk:              "",
		Status:            "failed",
		FlaggedQuestions:  []string{},
		FlaggedCandidates: 0,
		TotalAnalyzed:     0,
	})
	if err != nil {
		log.Error().Err(err).Str("driveId", driveID).Msg("Failed to update failed report")
	}
}

func validateComputePayload(req models.ComputeRequest) error {

	if req.DriveID == "" {
		return fmt.Errorf("driveId is required")
	}

	return nil
}
