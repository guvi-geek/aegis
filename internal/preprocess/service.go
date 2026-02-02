package preprocess

import (
	"context"
	"fmt"
	"time"

	"github.com/RishiKendai/aegis/internal/models"
	"github.com/RishiKendai/aegis/internal/repository"
)

type Service struct {
	client        *AstraClient
	artifactsRepo *repository.ArtifactsRepository
}

func NewService(client *AstraClient, artifactsRepo *repository.ArtifactsRepository) *Service {
	return &Service{
		client:        client,
		artifactsRepo: artifactsRepo,
	}
}

// processes a submission by calling Astra API and storing the result
func (s *Service) ProcessSubmission(ctx context.Context, submission *models.Submission) error {
	preprocessReq := &PreprocessRequest{
		EmailID:   submission.Email,
		AttemptID: submission.AttemptID,
		DriveID:   submission.DriveID,
		TestID:    submission.TestID,
		Code:      submission.SourceCode,
		Language:  submission.Language,
	}

	preprocessResp, err := s.client.Preprocess(ctx, preprocessReq)
	if err != nil {
		return fmt.Errorf("failed to preprocess: %w", err)
	}

	// Convert to artifact model
	artifact := &models.Artifact{
		Email:            preprocessResp.EmailID,
		AttemptID:        preprocessResp.AttemptID,
		TestID:           submission.TestID,
		DriveID:          submission.DriveID,
		Difficulty:       submission.Difficulty,
		SourceCode:       submission.SourceCode,
		QID:              submission.QID,
		Language:         preprocessResp.Language,
		LangCode:         submission.LangCode,
		Tokens:           preprocessResp.Preprocessing.Tokens,
		NormalizedTokens: preprocessResp.Preprocessing.NormalizedTokens,
		AST:              preprocessResp.Preprocessing.AST,
		CFG:              preprocessResp.Preprocessing.CFG,
		Fingerprints:     preprocessResp.Preprocessing.Fingerprints,
		CreatedAt:        time.Now(),
	}

	if err := s.artifactsRepo.InsertArtifact(ctx, artifact); err != nil {
		return fmt.Errorf("failed to store artifact: %w", err)
	}

	return nil
}
