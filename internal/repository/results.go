package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/RishiKendai/aegis/internal/models"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	resultsCollection = "results"
	reportsCollection = "plagiarism_reports"
)

type ResultsRepository struct {
	mongoRepo *MongoRepository
}

func NewResultsRepository(mongoRepo *MongoRepository) *ResultsRepository {
	return &ResultsRepository{
		mongoRepo: mongoRepo,
	}
}

func (r *ResultsRepository) InsertCandidateResult(ctx context.Context, result *models.CandidateResult) error {
	result.CreatedAt = time.Now()

	err := r.mongoRepo.InsertOne(ctx, resultsCollection, result)
	if err != nil {
		return fmt.Errorf("failed to insert candidate result: %w", err)
	}

	return nil
}

func (r *ResultsRepository) InsertTestReport(ctx context.Context, report *models.TestReport) error {
	report.CreatedAt = time.Now()

	err := r.mongoRepo.InsertOne(ctx, reportsCollection, report)
	if err != nil {
		return fmt.Errorf("failed to insert test report: %w", err)
	}

	return nil
}

func (r *ResultsRepository) UpdateTestReportByDriveID(ctx context.Context, driveID string, report *models.TestReport) error {
	filter := bson.M{"driveId": driveID}
	update := bson.M{
		"$set": bson.M{
			"risk":               report.Risk,
			"status":             report.Status,
			"flagged_qns":        report.FlaggedQuestions,
			"flagged_candidates": report.FlaggedCandidates,
			"total_analyzed":     report.TotalAnalyzed,
		},
	}

	_, err := r.mongoRepo.UpdateOne(ctx, reportsCollection, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update test report: %w", err)
	}

	return nil
}

func (r *ResultsRepository) GetLatestReportByDriveID(ctx context.Context, driveID string) (*models.TestReport, error) {
	filter := bson.M{"driveId": driveID}

	var report models.TestReport
	err := r.mongoRepo.FindOne(ctx, reportsCollection, filter).Decode(&report)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find report: %w", err)
	}

	return &report, nil
}

func (r *ResultsRepository) GetCandidateResultByDriveIDAndEmail(ctx context.Context, driveID, email string) (*models.CandidateResult, error) {
	filter := bson.M{"driveID": driveID, "email": email}
	opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	var result models.CandidateResult
	err := r.mongoRepo.FindOne(ctx, resultsCollection, filter, opts).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find candidate result: %w", err)
	}

	return &result, nil
}

func (r *ResultsRepository) UpdateCandidateResult(ctx context.Context, result *models.CandidateResult) error {
	log.Trace().
		Str("attemptID", result.AttemptID).
		Msg("here1")
	filter := bson.M{
		"attemptID": result.AttemptID,
		"driveId":   result.DriveID,
	}

	updateOps := bson.M{
		"$set": bson.M{
			"risk":              result.Risk,
			"code_similarity":   result.CodeSimilarity,
			"algo_similarity":   result.AlgoSimilarity,
			"plagiarism_status": result.PlagiarismStatus,
			"flagged_qns":       result.FlaggedQuestions,
			"plagiarism_peers":  result.PlagiarismPeers,
		},
	}
	updateResult, err := r.mongoRepo.UpdateOne(ctx, resultsCollection, filter, updateOps)
	if err != nil {
		log.Trace().
			Str("attemptID", result.AttemptID).
			Msg("Error")
		return fmt.Errorf("failed to update candidate result: %w", err)
	}

	// Check if document was found and updated
	if updateResult.MatchedCount == 0 {
		return fmt.Errorf("candidate result not found for attemptId: %s, driveId: %s", result.AttemptID, result.DriveID)
	}

	return nil
}
