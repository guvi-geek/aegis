package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/RishiKendai/aegis/internal/config"
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
	computeSem     chan struct{} // Semaphore for bounded concurrency
	computeTimeout time.Duration
}

// NewHandler creates a new handler
func NewHandler(
	cfg *config.Config,
	artifactsRepo *repository.ArtifactsRepository,
	resultsRepo *repository.ResultsRepository,
	workerPool *plagiarism.WorkerPool,
) *Handler {
	// Create semaphore for bounded concurrency
	sem := make(chan struct{}, cfg.MaxConcurrentCompute)

	return &Handler{
		cfg:            cfg,
		artifactsRepo:  artifactsRepo,
		resultsRepo:    resultsRepo,
		workerPool:     workerPool,
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
		c.JSON(http.StatusConflict, ErrorResponse{
			Error: "Computation already completed for this driveId",
			Code:  "COMPUTATION_COMPLETED",
		})
		return
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

	// Return 202 Accepted immediately
	c.JSON(http.StatusAccepted, models.ComputeResponse{
		Message: "Plagiarism computation started",
		TestID:  req.DriveID,
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
		DriveID:   driveID,
		Risk:      "",
		Status:    "pending",
		FlaggedQN: []string{},
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
	failedReport := &models.TestReport{
		DriveID:   driveID,
		Risk:      "",
		Status:    "failed",
		FlaggedQN: []string{},
	}

	if err := h.resultsRepo.InsertTestReport(ctx, failedReport); err != nil {
		log.Error().Err(err).Str("driveId", driveID).Msg("Failed to create failed report")
	}
}

func validateComputePayload(req models.ComputeRequest) error {

	if req.DriveID == "" {
		return fmt.Errorf("driveId is required")
	}

	return nil
}
