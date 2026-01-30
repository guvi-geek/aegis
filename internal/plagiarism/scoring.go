package plagiarism

import (
	"math"
	"sort"

	"github.com/RishiKendai/aegis/internal/models"
	"github.com/rs/zerolog/log"
)

const (
	SignificantSimilarityThreshold = 0.55

	AlgorithmicSimilarityThreshold = 0.70

	// Risk level constants for candidate results (standardized format)
	RiskClean            = "clean"
	RiskSuspicious       = "suspicious"
	RiskHighlySuspicious = "highly_suspicious"
	RiskNearCopy         = "near_copy"

	// Risk level constants for test reports (standardized format)
	TestRiskSafe     = "safe"
	TestRiskModerate = "moderate"
	TestRiskHigh     = "high"
	TestRiskCritical = "critical"
)

// PairSimilarity represents similarity between a pair of artifacts
type PairSimilarity struct {
	ArtifactA  *models.Artifact
	ArtifactB  *models.Artifact
	FinalScore float64
	QID        string
	Difficulty string
}

// CandidateScore calculates candidate score using Top-K + boost formula
func CandidateScore(pairs []PairSimilarity) float64 {
	significantPairs := make([]PairSimilarity, 0)
	for _, pair := range pairs {
		if pair.FinalScore >= SignificantSimilarityThreshold {
			significantPairs = append(significantPairs, pair)
		}
	}

	// If no significant pairs, return 0
	if len(significantPairs) == 0 {
		return 0.0
	}

	// Step 2: Take top K=3 scores
	K := 3
	if len(significantPairs) < K {
		K = len(significantPairs)
	}

	// Sort by score descending
	sort.Slice(significantPairs, func(i, j int) bool {
		return significantPairs[i].FinalScore > significantPairs[j].FinalScore
	})

	// Step 3: Calculate average of top K scores
	topScores := significantPairs[:K]
	sum := 0.0
	for _, pair := range topScores {
		sum += pair.FinalScore
	}
	candidateScore := sum / float64(K)

	// Step 4: Frequency boost
	// M = number of distinct candidates with FinalScore >= SignificantSimilarityThreshold
	distinctCandidates := make(map[string]bool)
	for _, pair := range significantPairs {
		distinctCandidates[pair.ArtifactB.Email] = true
	}
	M := len(distinctCandidates)

	boost := math.Min(0.15, 0.05*float64(M-1))
	if M == 0 {
		boost = 0.0
	}

	candidateScore += boost

	// Clamp to [0, 1]
	if candidateScore > 1.0 {
		candidateScore = 1.0
	}
	if candidateScore < 0.0 {
		candidateScore = 0.0
	}
	log.Trace().
		Float64("candidateScore", candidateScore).
		Msg("Candidate score")
	return candidateScore
}

// GetRiskLevel returns risk level based on candidate score
func GetRiskLevel(score float64) string {
	if score < 0.3 {
		return RiskClean
	} else if score < 0.6 {
		return RiskSuspicious
	} else if score < 0.85 {
		return RiskHighlySuspicious
	}
	return RiskNearCopy
}

// TestRisk calculates test risk using the formula
func TestRisk(totalQuestions int, avgDifficulty float64, avgSimilarity float64, flaggedQuestions int) (float64, string) {
	Q := float64(totalQuestions)
	D := avgDifficulty // 0..1 (EASY=0.33, MEDIUM=0.66, HARD=1.0)
	BASE := 0.70

	// Calculate threshold
	questionFactor := 1.0 / math.Sqrt(Q)
	difficultyFactor := 0.5 + D
	threshold := BASE * questionFactor * difficultyFactor
	threshold = math.Max(0.35, math.Min(0.85, threshold)) // Clamp [0.35, 0.85]

	// Calculate risk
	S := avgSimilarity
	R := float64(flaggedQuestions)
	risk := (0.7 * S) + (0.3 * (R / Q))

	// Determine risk level
	var riskLevel string
	if risk < 0.40 {
		riskLevel = TestRiskSafe
	} else if risk < 0.60 {
		riskLevel = TestRiskModerate
	} else if risk < 0.80 {
		riskLevel = TestRiskHigh
	} else {
		riskLevel = TestRiskCritical
	}

	return risk, riskLevel
}

// DifficultyToFloat converts difficulty string to float (0..1)
func DifficultyToFloat(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.33
	case "medium":
		return 0.66
	case "hard":
		return 1.0
	default:
		return 0.5 // Default to medium
	}
}
