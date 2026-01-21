package plagiarism

import (
	"context"
	"fmt"
	"strconv"

	"github.com/RishiKendai/aegis/internal/models"
	"github.com/RishiKendai/aegis/internal/repository"
	"github.com/rs/zerolog/log"
)

// ComputationJob represents a job for the worker pool
type ComputationJob struct {
	Pair       Pair
	Difficulty string
	QID        string
	ResultChan chan<- PairSimilarity
	DoneChan   chan<- struct{}
}

// Execute executes the computation job
func (j *ComputationJob) Execute(ctx context.Context) error {
	defer func() {
		// Signal completion
		select {
		case j.DoneChan <- struct{}{}:
		default:
		}
	}()

	result := CascadePipeline(j.Pair.ArtifactA, j.Pair.ArtifactB, j.Difficulty)

	pairSimilarity := PairSimilarity{
		ArtifactA:  j.Pair.ArtifactA,
		ArtifactB:  j.Pair.ArtifactB,
		FinalScore: result.FinalScore,
		QID:        j.QID,
		Difficulty: j.Difficulty,
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case j.ResultChan <- pairSimilarity:
		return nil
	}
}

func ComputePlagiarism(
	ctx context.Context,
	driveID string,
	artifactsRepo *repository.ArtifactsRepository,
	resultsRepo *repository.ResultsRepository,
	workerPool *WorkerPool,
	batchSize int,
) error {
	// Load all artifacts for driveId
	artifacts, err := artifactsRepo.GetArtifactsByDriveID(ctx, driveID)
	if err != nil {
		log.Error().Err(err).Str("driveId", driveID).Msg("Failed to load artifacts")
		return fmt.Errorf("failed to load artifacts: %w", err)
	}

	// Edge Case: Missing driveId
	if len(artifacts) == 0 {
		return fmt.Errorf("no artifacts found for driveId: %s", driveID)
	}

	// Edge Case: Single candidate
	uniqueCandidates := make(map[string]bool)
	for _, artifact := range artifacts {
		uniqueCandidates[artifact.Email] = true
	}
	if len(uniqueCandidates) == 1 {
		return handleSingleCandidate(ctx, artifacts[0], resultsRepo, driveID)
	}

	// Group by qId, then by language
	buckets := groupByQuestionAndLanguage(artifacts)

	// Process each bucket
	allPairSimilarities := make([]PairSimilarity, 0)
	candidatePairsMap := make(map[string][]PairSimilarity) // email -> []PairSimilarity

	for qID, langBuckets := range buckets {
		for language, bucketArtifacts := range langBuckets {
			if len(bucketArtifacts) < 2 {
				// Edge Case: No pairs possible in this bucket
				continue
			}

			// Build GII (optimization: skip hashes with only 1 candidate)
			gii := BuildGII(bucketArtifacts)

			// Edge Case: No worthy pairs
			if len(gii) == 0 {
				log.Info().
					Str("qId", qID).
					Str("language", language).
					Msg("No worthy pairs found (GII empty)")
				continue
			}

			difficulty := bucketArtifacts[0].Difficulty

			// Find worthy pairs
			worthyPairs := GetWorthyPairs(gii, bucketArtifacts, difficulty)

			if len(worthyPairs) == 0 {
				log.Info().
					Str("qId", qID).
					Str("language", language).
					Msg("No worthy pairs found after threshold check")
				continue
			}

			// Process pairs in batches
			pairSimilarities := processPairsInBatches(
				ctx,
				worthyPairs,
				difficulty,
				qID,
				workerPool,
				batchSize,
			)

			// Filter pairs with FinalScore >= 0.55 (significant pairs)
			significantPairs := make([]PairSimilarity, 0)
			for _, ps := range pairSimilarities {
				if ps.FinalScore >= 0.55 {
					significantPairs = append(significantPairs, ps)
					// Track pairs for each candidate
					candidatePairsMap[ps.ArtifactA.Email] = append(candidatePairsMap[ps.ArtifactA.Email], ps)
					candidatePairsMap[ps.ArtifactB.Email] = append(candidatePairsMap[ps.ArtifactB.Email], ps)
				}
			}

			allPairSimilarities = append(allPairSimilarities, significantPairs...)
		}
	}

	// Edge Case: Short-circuit stops (no pairs with FinalScore >= 0.55)
	if len(allPairSimilarities) == 0 {
		return handleNoSignificantPairs(ctx, artifacts, resultsRepo, driveID)
	}

	// Aggregate results
	return aggregateResults(ctx, artifacts, allPairSimilarities, candidatePairsMap, resultsRepo, driveID)
}

// processPairsInBatches processes pairs in batches
func processPairsInBatches(
	ctx context.Context,
	pairs []Pair,
	difficulty string,
	qID string,
	workerPool *WorkerPool,
	batchSize int,
) []PairSimilarity {
	resultChan := make(chan PairSimilarity, len(pairs))
	doneChan := make(chan struct{}, len(pairs))

	// Submit all jobs
	for _, pair := range pairs {
		job := &ComputationJob{
			Pair:       pair,
			Difficulty: difficulty,
			QID:        qID,
			ResultChan: resultChan,
			DoneChan:   doneChan,
		}

		if err := workerPool.Submit(job); err != nil {
			log.Error().Err(err).Msg("Failed to submit job")
		}
	}

	// Collect results as jobs complete
	expectedResults := len(pairs)
	resultsMap := make(map[string]PairSimilarity) // Use pair key to track results

	for len(resultsMap) < expectedResults {
		select {
		case <-ctx.Done():
			// Return what we have so far
			finalResults := make([]PairSimilarity, 0, len(resultsMap))
			for _, result := range resultsMap {
				finalResults = append(finalResults, result)
			}
			return finalResults
		case result := <-resultChan:
			// Use pair key to avoid duplicates
			pairKey := getPairKey(result.ArtifactA.AttemptID, result.ArtifactB.AttemptID)
			resultsMap[pairKey] = result
		case <-doneChan:
			// Job completed, continue waiting for results
		}
	}

	// Convert map to slice
	finalResults := make([]PairSimilarity, 0, len(resultsMap))
	for _, result := range resultsMap {
		finalResults = append(finalResults, result)
	}

	return finalResults
}

// groupByQuestionAndLanguage groups artifacts by qId and language
func groupByQuestionAndLanguage(artifacts []*models.Artifact) map[string]map[string][]*models.Artifact {
	buckets := make(map[string]map[string][]*models.Artifact)

	for _, artifact := range artifacts {
		qID := artifact.QID
		language := artifact.Language

		if buckets[strconv.FormatInt(qID, 10)] == nil {
			buckets[strconv.FormatInt(qID, 10)] = make(map[string][]*models.Artifact)
		}
		buckets[strconv.FormatInt(qID, 10)][language] = append(buckets[strconv.FormatInt(qID, 10)][language], artifact)
	}

	return buckets
}

// handleSingleCandidate handles the case when there's only one candidate
func handleSingleCandidate(
	ctx context.Context,
	artifact *models.Artifact,
	resultsRepo *repository.ResultsRepository,
	driveID string,
) error {
	candidateResult := &models.CandidateResult{
		Email:           artifact.Email,
		AttemptID:       artifact.AttemptID,
		DriveID:         driveID,
		Risk:            "clean",
		FlaggedQN:       []string{},
		PlagiarismPeers: make(map[string][]string),
		CodeSimilarity:  0,
		AlgoSimilarity:  0,
		Status:          "completed",
	}

	if err := resultsRepo.InsertCandidateResult(ctx, candidateResult); err != nil {
		return fmt.Errorf("failed to insert candidate result: %w", err)
	}

	testReport := &models.TestReport{
		DriveID:   driveID,
		Risk:      "Safe",
		Status:    "completed",
		FlaggedQN: []string{},
	}

	if err := resultsRepo.InsertTestReport(ctx, testReport); err != nil {
		return fmt.Errorf("failed to insert test report: %w", err)
	}

	log.Debug().
		Str("driveId", driveID).
		Msg("Handled single candidate case")

	return nil
}

// handleNoSignificantPairs handles the case when no pairs pass the threshold
func handleNoSignificantPairs(
	ctx context.Context,
	artifacts []*models.Artifact,
	resultsRepo *repository.ResultsRepository,
	driveID string,
) error {
	// Create "clean" results for all candidates
	uniqueCandidates := make(map[string]*models.Artifact)
	for _, artifact := range artifacts {
		if _, exists := uniqueCandidates[artifact.Email]; !exists {
			uniqueCandidates[artifact.Email] = artifact
		}
	}

	for _, artifact := range uniqueCandidates {
		candidateResult := &models.CandidateResult{
			Email:           artifact.Email,
			AttemptID:       artifact.AttemptID,
			DriveID:         driveID,
			Risk:            "clean",
			FlaggedQN:       []string{},
			PlagiarismPeers: make(map[string][]string),
			CodeSimilarity:  0,
			AlgoSimilarity:  0,
			Status:          "completed",
		}

		if err := resultsRepo.InsertCandidateResult(ctx, candidateResult); err != nil {
			return fmt.Errorf("failed to insert candidate result: %w", err)
		}
	}

	testReport := &models.TestReport{
		DriveID:   driveID,
		Risk:      "Safe",
		Status:    "completed",
		FlaggedQN: []string{},
	}

	if err := resultsRepo.InsertTestReport(ctx, testReport); err != nil {
		return fmt.Errorf("failed to insert test report: %w", err)
	}

	log.Info().
		Str("driveId", driveID).
		Msg("Handled no significant pairs case")

	return nil
}

// aggregateResults aggregates results and creates candidate and test reports
func aggregateResults(
	ctx context.Context,
	artifacts []*models.Artifact,
	allPairSimilarities []PairSimilarity,
	candidatePairsMap map[string][]PairSimilarity,
	resultsRepo *repository.ResultsRepository,
	driveID string,
) error {
	// Get unique candidates
	uniqueCandidates := make(map[string]*models.Artifact)
	for _, artifact := range artifacts {
		if _, exists := uniqueCandidates[artifact.Email]; !exists {
			uniqueCandidates[artifact.Email] = artifact
		}
	}

	// Calculate candidate scores
	candidateResults := make([]*models.CandidateResult, 0)
	flaggedQNs := make(map[string]bool)
	flaggedCandidates := 0

	for email, artifact := range uniqueCandidates {
		pairs := candidatePairsMap[email]
		if len(pairs) == 0 {
			// No significant pairs for this candidate
			candidateResult := &models.CandidateResult{
				Email:           email,
				AttemptID:       artifact.AttemptID,
				DriveID:         driveID,
				Risk:            "clean",
				FlaggedQN:       []string{},
				PlagiarismPeers: make(map[string][]string),
				CodeSimilarity:  0,
				AlgoSimilarity:  0,
				Status:          "completed",
			}
			candidateResults = append(candidateResults, candidateResult)
			continue
		}

		// Calculate candidate score
		score := CandidateScore(pairs)
		risk := GetRiskLevel(score)

		// Build flagged questions and plagiarism peers
		flaggedQNSet := make(map[string]bool)
		plagiarismPeers := make(map[string][]string)
		codeSimilarity := 0
		algoSimilarity := 0

		for _, pair := range pairs {
			flaggedQNSet[pair.QID] = true

			if _, exists := plagiarismPeers[pair.QID]; !exists {
				plagiarismPeers[pair.QID] = make([]string, 0)
			}

			// Add peer
			if pair.ArtifactA.Email == email {
				plagiarismPeers[pair.QID] = append(plagiarismPeers[pair.QID], pair.ArtifactB.AttemptID)
			} else {
				plagiarismPeers[pair.QID] = append(plagiarismPeers[pair.QID], pair.ArtifactA.AttemptID)
			}

			// Count similarities
			if pair.FinalScore >= 0.55 {
				codeSimilarity++
			}
			if pair.FinalScore >= 0.70 {
				algoSimilarity++
			}
		}

		flaggedQNList := make([]string, 0, len(flaggedQNSet))
		for qID := range flaggedQNSet {
			flaggedQNList = append(flaggedQNList, qID)
			flaggedQNs[qID] = true
		}

		if risk != "clean" {
			flaggedCandidates++
		}

		candidateResult := &models.CandidateResult{
			Email:           email,
			AttemptID:       artifact.AttemptID,
			DriveID:         driveID,
			Risk:            risk,
			FlaggedQN:       flaggedQNList,
			PlagiarismPeers: plagiarismPeers,
			CodeSimilarity:  codeSimilarity,
			AlgoSimilarity:  algoSimilarity,
			Status:          "completed",
		}

		candidateResults = append(candidateResults, candidateResult)
	}

	// Insert candidate results
	for _, result := range candidateResults {
		if err := resultsRepo.InsertCandidateResult(ctx, result); err != nil {
			return fmt.Errorf("failed to insert candidate result: %w", err)
		}
	}

	// Calculate test risk
	totalQuestions := len(groupByQuestionAndLanguage(artifacts))

	// Calculate average difficulty
	avgDifficulty := 0.0
	difficultyCount := 0
	for _, artifact := range artifacts {
		avgDifficulty += DifficultyToFloat(artifact.Difficulty)
		difficultyCount++
	}
	if difficultyCount > 0 {
		avgDifficulty /= float64(difficultyCount)
	}

	// Calculate average similarity
	avgSimilarity := 0.0
	if len(allPairSimilarities) > 0 {
		sum := 0.0
		for _, ps := range allPairSimilarities {
			sum += ps.FinalScore
		}
		avgSimilarity = sum / float64(len(allPairSimilarities))
	}

	flaggedQNList := make([]string, 0, len(flaggedQNs))
	for qID := range flaggedQNs {
		flaggedQNList = append(flaggedQNList, qID)
	}

	_, riskLevel := TestRisk(totalQuestions, avgDifficulty, avgSimilarity, len(flaggedQNList))

	testReport := &models.TestReport{
		DriveID:   driveID,
		Risk:      riskLevel,
		Status:    "completed",
		FlaggedQN: flaggedQNList,
	}

	if err := resultsRepo.InsertTestReport(ctx, testReport); err != nil {
		return fmt.Errorf("failed to insert test report: %w", err)
	}

	log.Info().
		Str("driveId", driveID).
		Int("candidates", len(candidateResults)).
		Int("flagged", flaggedCandidates).
		Str("testRisk", riskLevel).
		Msg("Computation completed successfully")

	return nil
}
