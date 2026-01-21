package plagiarism

import (
	"github.com/RishiKendai/aegis/internal/models"
)

// GII (Global Inverted Index) maps hash → [submission_ids]
type GII map[string][]string

// BuildGII builds Global Inverted Index from artifacts
// Optimization: Skip hashes that appear in only 1 candidate
func BuildGII(artifacts []*models.Artifact) GII {
	gii := make(GII)

	// First pass: Build hash → [attemptIds] mapping
	for _, artifact := range artifacts {
		if artifact.Fingerprints == nil {
			continue
		}

		attemptID := artifact.AttemptID
		for _, hashEntry := range artifact.Fingerprints.Hashes {
			hash := hashEntry.Hash
			gii[hash] = append(gii[hash], attemptID)
		}
	}

	// Second pass: Filter out hashes with only 1 candidate (optimization)
	filteredGII := make(GII)
	for hash, attemptIDs := range gii {
		if len(attemptIDs) >= 2 {
			// Only include hashes with 2+ candidates
			filteredGII[hash] = attemptIDs
		}
	}

	return filteredGII
}

// GetWorthyPairs finds worthy pairs based on difficulty threshold
func GetWorthyPairs(gii GII, artifacts []*models.Artifact, difficulty string) []Pair {
	// Build artifact map for quick lookup
	artifactMap := make(map[string]*models.Artifact)
	for _, artifact := range artifacts {
		artifactMap[artifact.AttemptID] = artifact
	}

	// Get threshold based on difficulty
	threshold := getWorthyThreshold(difficulty)

	// Build pair map to avoid duplicates
	pairMap := make(map[string]Pair)

	// For each hash with 2+ candidates, find pairs
	for _, attemptIDs := range gii {
		if len(attemptIDs) < 2 {
			continue // Skip (shouldn't happen after filtering, but safety check)
		}

		// Get artifacts for this hash
		hashArtifacts := make([]*models.Artifact, 0)
		for _, attemptID := range attemptIDs {
			if artifact, ok := artifactMap[attemptID]; ok {
				hashArtifacts = append(hashArtifacts, artifact)
			}
		}

		// Calculate shared hashes for each pair
		for i := 0; i < len(hashArtifacts); i++ {
			for j := i + 1; j < len(hashArtifacts); j++ {
				artifactA := hashArtifacts[i]
				artifactB := hashArtifacts[j]

				// Calculate overlap
				overlap := calculateOverlap(artifactA, artifactB)
				if overlap >= threshold {
					// Create pair key (sorted to avoid duplicates)
					pairKey := getPairKey(artifactA.AttemptID, artifactB.AttemptID)
					if _, exists := pairMap[pairKey]; !exists {
						pairMap[pairKey] = Pair{
							ArtifactA: artifactA,
							ArtifactB: artifactB,
						}
					}
				}
			}
		}
	}

	// Convert map to slice
	worthyPairs := make([]Pair, 0, len(pairMap))
	for _, pair := range pairMap {
		worthyPairs = append(worthyPairs, pair)
	}

	return worthyPairs
}

// calculateOverlap calculates shared_hashes / min(total_hashes_A, total_hashes_B)
func calculateOverlap(artifactA, artifactB *models.Artifact) float64 {
	if artifactA.Fingerprints == nil || artifactB.Fingerprints == nil {
		return 0.0
	}

	hashesA := make(map[string]bool)
	for _, hashEntry := range artifactA.Fingerprints.Hashes {
		hashesA[hashEntry.Hash] = true
	}

	hashesB := make(map[string]bool)
	for _, hashEntry := range artifactB.Fingerprints.Hashes {
		hashesB[hashEntry.Hash] = true
	}

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

	minTotal := totalA
	if totalB < minTotal {
		minTotal = totalB
	}

	if minTotal == 0 {
		return 0.0
	}

	return float64(sharedCount) / float64(minTotal)
}

// getWorthyThreshold returns threshold based on difficulty
func getWorthyThreshold(difficulty string) float64 {
	switch difficulty {
	case "easy":
		return 0.15 // 15%
	case "medium":
		return 0.10 // 10%
	case "hard":
		return 0.05 // 5%
	default:
		return 0.10 // Default to medium
	}
}

// Pair represents a pair of artifacts to compare
type Pair struct {
	ArtifactA *models.Artifact
	ArtifactB *models.Artifact
}

// getPairKey creates a sorted key for a pair to avoid duplicates
func getPairKey(id1, id2 string) string {
	if id1 < id2 {
		return id1 + ":" + id2
	}
	return id2 + ":" + id1
}
