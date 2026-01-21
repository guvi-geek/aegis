package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/RishiKendai/aegis/internal/models"
	"go.mongodb.org/mongo-driver/bson"
)

const artifactsCollection = "plagiarism_artifacts"

type ArtifactsRepository struct {
	mongoRepo *MongoRepository
}

func NewArtifactsRepository(mongoRepo *MongoRepository) *ArtifactsRepository {
	return &ArtifactsRepository{
		mongoRepo: mongoRepo,
	}
}

func (r *ArtifactsRepository) InsertArtifact(ctx context.Context, artifact *models.Artifact) error {
	artifact.CreatedAt = time.Now()
	err := r.mongoRepo.InsertOne(ctx, artifactsCollection, artifact)
	if err != nil {
		return fmt.Errorf("failed to insert artifact: %w", err)
	}

	return nil
}

func (r *ArtifactsRepository) GetArtifactsByDriveID(ctx context.Context, driveID string) ([]*models.Artifact, error) {
	filter := bson.M{"driveId": driveID}

	cursor, err := r.mongoRepo.FindMany(ctx, artifactsCollection, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find artifacts: %w", err)
	}
	defer cursor.Close(ctx)

	var artifacts []*models.Artifact
	if err := cursor.All(ctx, &artifacts); err != nil {
		return nil, fmt.Errorf("failed to decode artifacts: %w", err)
	}

	return artifacts, nil
}

func (r *ArtifactsRepository) GetArtifactsByDriveIDAndQID(ctx context.Context, driveID, qID string) ([]*models.Artifact, error) {
	filter := bson.M{"driveId": driveID, "qId": qID}

	cursor, err := r.mongoRepo.FindMany(ctx, artifactsCollection, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find artifacts: %w", err)
	}
	defer cursor.Close(ctx)

	var artifacts []*models.Artifact
	if err := cursor.All(ctx, &artifacts); err != nil {
		return nil, fmt.Errorf("failed to decode artifacts: %w", err)
	}

	return artifacts, nil
}

func (r *ArtifactsRepository) CountArtifactsByDriveID(ctx context.Context, driveID string) (int64, error) {
	filter := bson.M{"driveId": driveID}

	count, err := r.mongoRepo.CountDocuments(ctx, artifactsCollection, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count artifacts: %w", err)
	}

	return count, nil
}
