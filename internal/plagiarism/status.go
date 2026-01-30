package plagiarism

import (
	"context"
	"fmt"
	"time"

	"github.com/RishiKendai/aegis/internal/infra/redis"
	"github.com/RishiKendai/aegis/internal/models"
	"github.com/rs/zerolog/log"
)

func UpdateStatus(ctx context.Context, redisClient *redis.Client, driveID string, step models.Step) error {
	validSteps := map[models.Step]bool{
		models.StepIdle:          true,
		models.StepInitiated:     true,
		models.StepStarted:       true,
		models.StepPreprocessing: true,
		models.StepFiltering:     true,
		models.StepDeepAnalysis:  true,
		models.StepCompleted:     true,
	}
	if !validSteps[step] {
		return fmt.Errorf("unknown step: %s", step)
	}

	rkey := "plagiarism_report_status:" + driveID

	err := redisClient.Set(ctx, rkey, string(step), 12*time.Hour).Err()
	if err != nil {
		log.Error().Err(err).
			Str("step", string(step)).
			Str("driveID", driveID).
			Str("redisKey", rkey).
			Msg("Failed to update status in Redis")
		return fmt.Errorf("failed to update status in Redis: %w", err)
	}

	log.Trace().
		Str("step", string(step)).
		Msg("✌️Status updated in Redis")

	return nil
}
