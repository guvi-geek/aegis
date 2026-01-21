package plagiarism

import (
	"math"
	"sort"

	"github.com/RishiKendai/aegis/internal/models"
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
	// Step 1: Filter pairs where FinalScore >= 0.55
	significantPairs := make([]PairSimilarity, 0)
	for _, pair := range pairs {
		if pair.FinalScore >= 0.55 {
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
	// M = number of distinct candidates with FinalScore >= 0.55
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

	return candidateScore
}

// GetRiskLevel returns risk level based on candidate score
func GetRiskLevel(score float64) string {
	if score < 0.3 {
		return "clean"
	} else if score < 0.6 {
		return "suspicious"
	} else if score < 0.85 {
		return "highly suspicious"
	}
	return "Near copy"
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
		riskLevel = "Safe"
	} else if risk < 0.60 {
		riskLevel = "Moderate"
	} else if risk < 0.80 {
		riskLevel = "High"
	} else {
		riskLevel = "Critical"
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
