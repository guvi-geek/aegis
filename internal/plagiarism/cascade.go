package plagiarism

import (
	"github.com/RishiKendai/aegis/internal/models"
)

// SimilarityScores holds scores from all algorithms
type SimilarityScores struct {
	Fingerprint float64
	Token       float64
	AST         float64
	CFG         float64
}

// CascadeResult holds the result of cascade pipeline
type CascadeResult struct {
	Scores         SimilarityScores
	ShortCircuited bool
	FinalScore     float64
}

// CascadePipeline implements progressive short-circuit pipeline
// Order: Fingerprint → Token → AST → CFG
func CascadePipeline(artifactA, artifactB *models.Artifact, difficulty string) *CascadeResult {
	result := &CascadeResult{
		Scores: SimilarityScores{},
	}

	// Get weights based on difficulty
	weights := getWeights(difficulty)

	// Initialize accumulated score
	currentScore := 0.0
	remainingMax := weights.Fingerprint + weights.Token + weights.AST + weights.CFG

	// 1. Fingerprint (coarsest, fastest)
	result.Scores.Fingerprint = FingerprintSimilarity(artifactA, artifactB)
	currentScore += result.Scores.Fingerprint * weights.Fingerprint
	remainingMax -= weights.Fingerprint
	layerThreshold := getThreshold(difficulty, "fingerprint")

	if shouldShortCircuit(currentScore, remainingMax, layerThreshold) {
		result.ShortCircuited = true
		result.FinalScore = currentScore
		return result
	}

	// 2. Token (GST)
	result.Scores.Token = TokenSimilarity(artifactA, artifactB)
	currentScore += result.Scores.Token * weights.Token
	remainingMax -= weights.Token
	layerThreshold = getThreshold(difficulty, "token")

	if shouldShortCircuit(currentScore, remainingMax, layerThreshold) {
		result.ShortCircuited = true
		result.FinalScore = currentScore
		return result
	}

	// 3. AST (Merkle)
	result.Scores.AST = ASTSimilarity(artifactA, artifactB)
	currentScore += result.Scores.AST * weights.AST
	remainingMax -= weights.AST
	layerThreshold = getThreshold(difficulty, "ast")

	if shouldShortCircuit(currentScore, remainingMax, layerThreshold) {
		result.ShortCircuited = true
		result.FinalScore = currentScore
		return result
	}

	// 4. CFG (finest, slowest)
	result.Scores.CFG = CFGSimilarity(artifactA, artifactB)
	currentScore += result.Scores.CFG * weights.CFG
	result.FinalScore = currentScore

	return result
}

func shouldShortCircuit(currentScore, remainingMax, threshold float64) bool {
	// If current weighted score + max possible from remaining < threshold, short-circuit
	return currentScore+remainingMax < threshold
}

// Weights holds algorithm weights by difficulty
type Weights struct {
	Fingerprint float64
	Token       float64
	AST         float64
	CFG         float64
}

// getWeights returns weights based on difficulty
func getWeights(difficulty string) Weights {
	switch difficulty {
	case "easy":
		return Weights{
			Fingerprint: 0.50,
			Token:       0.30,
			AST:         0.15,
			CFG:         0.05,
		}
	case "medium":
		return Weights{
			Fingerprint: 0.40,
			Token:       0.30,
			AST:         0.20,
			CFG:         0.10,
		}
	case "hard":
		return Weights{
			Fingerprint: 0.30,
			Token:       0.25,
			AST:         0.30,
			CFG:         0.15,
		}
	default:
		// Default to medium
		return Weights{
			Fingerprint: 0.40,
			Token:       0.30,
			AST:         0.20,
			CFG:         0.10,
		}
	}
}

func getThreshold(difficulty, layer string) float64 {
	switch difficulty {
	case "easy":
		switch layer {
		case "fingerprint":
			return 0.65 // High threshold: easy problem + coarse layer = high false positive rate
		case "token":
			return 0.60
		case "ast":
			return 0.58
		case "cfg":
			return 0.55 // Lower threshold for CFG (more accurate layer)
		default:
			return 0.55 // Default fallback
		}
	case "medium":
		switch layer {
		case "fingerprint":
			return 0.58 // Medium-high threshold for fingerprint
		case "token":
			return 0.55
		case "ast":
			return 0.52
		case "cfg":
			return 0.50 // Lower threshold for CFG
		default:
			return 0.55 // Default fallback
		}
	case "hard":
		switch layer {
		case "fingerprint":
			return 0.52 // Medium threshold: hard problem reduces false positives even for coarse layer
		case "token":
			return 0.50
		case "ast":
			return 0.48
		case "cfg":
			return 0.45 // Low threshold: hard problem + fine-grained layer = need to catch subtle similarities
		default:
			return 0.55 // Default fallback
		}
	default:
		// Default to medium thresholds
		switch layer {
		case "fingerprint":
			return 0.58
		case "token":
			return 0.55
		case "ast":
			return 0.52
		case "cfg":
			return 0.50
		default:
			return 0.55
		}
	}
}
