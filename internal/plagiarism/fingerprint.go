package plagiarism

import (
	"github.com/RishiKendai/aegis/internal/models"
)

// FingerprintSimilarity calculates similarity using pre-computed fingerprints
func FingerprintSimilarity(artifactA, artifactB *models.Artifact) float64 {
	if artifactA.Fingerprints == nil || artifactB.Fingerprints == nil {
		return 0.0
	}

	// Build hash sets for fast lookup
	hashesA := make(map[string]bool)
	for _, hashEntry := range artifactA.Fingerprints.Hashes {
		hashesA[hashEntry.Hash] = true
	}

	hashesB := make(map[string]bool)
	for _, hashEntry := range artifactB.Fingerprints.Hashes {
		hashesB[hashEntry.Hash] = true
	}

	// Count shared hashes
	sharedCount := 0
	for hash := range hashesA {
		if hashesB[hash] {
			sharedCount++
		}
	}

	totalA := len(hashesA)
	totalB := len(hashesB)

	if totalA == 0 || totalB == 0 {
		return 0.0
	}

	// FP_Score = shared_hashes / min(total_hashes_A, total_hashes_B)
	minTotal := totalA
	if totalB < minTotal {
		minTotal = totalB
	}

	if minTotal == 0 {
		return 0.0
	}

	return float64(sharedCount) / float64(minTotal)
}
