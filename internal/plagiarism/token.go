package plagiarism

import (
	"github.com/RishiKendai/aegis/internal/models"
)

// TokenSimilarity calculates similarity using Greedy String Tiling (GST)
func TokenSimilarity(artifactA, artifactB *models.Artifact) float64 {
	tokensA := artifactA.NormalizedTokens
	tokensB := artifactB.NormalizedTokens

	if len(tokensA) == 0 || len(tokensB) == 0 {
		return 0.0
	}

	// Find maximal common token substrings (min length ≥ 5)
	matchedTokens := greedyStringTiling(tokensA, tokensB, 5)

	// TokenScore = 2 * matched_tokens / (lenA + lenB)
	totalLen := len(tokensA) + len(tokensB)
	if totalLen == 0 {
		return 0.0
	}

	return 2.0 * float64(matchedTokens) / float64(totalLen)
}

// greedyStringTiling implements the Greedy String Tiling algorithm
// Returns the number of matched tokens
func greedyStringTiling(tokensA, tokensB []string, minLength int) int {
	matched := make([]bool, len(tokensA))
	matchedB := make([]bool, len(tokensB))
	totalMatched := 0

	for {
		maxMatch := 0
		maxStartA := -1
		maxStartB := -1

		// Find maximal common substring
		for i := 0; i < len(tokensA); i++ {
			if matched[i] {
				continue
			}

			for j := 0; j < len(tokensB); j++ {
				if matchedB[j] {
					continue
				}

				// Try to match from this position
				matchLen := 0
				for k := 0; i+k < len(tokensA) && j+k < len(tokensB); k++ {
					if matched[i+k] || matchedB[j+k] {
						break
					}
					if tokensA[i+k] != tokensB[j+k] {
						break
					}
					matchLen++
				}

				if matchLen >= minLength && matchLen > maxMatch {
					maxMatch = matchLen
					maxStartA = i
					maxStartB = j
				}
			}
		}

		if maxMatch == 0 {
			break // No more matches
		}

		// Mark matched tokens
		for k := 0; k < maxMatch; k++ {
			matched[maxStartA+k] = true
			matchedB[maxStartB+k] = true
			totalMatched++
		}
	}

	return totalMatched
}
