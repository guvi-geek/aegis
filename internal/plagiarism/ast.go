package plagiarism

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/RishiKendai/aegis/internal/models"
)

// ASTSimilarity calculates similarity using AST Merkle hashing
func ASTSimilarity(artifactA, artifactB *models.Artifact) float64 {
	if artifactA.AST == nil || artifactB.AST == nil {
		return 0.0
	}

	// Build multiset of subtree hashes for both ASTs
	subtreesA := buildSubtreeHashes(artifactA.AST)
	subtreesB := buildSubtreeHashes(artifactB.AST)

	// Count common subtrees
	commonCount := 0
	for hash := range subtreesA {
		if subtreesB[hash] {
			commonCount++
		}
	}

	totalA := len(subtreesA)
	totalB := len(subtreesB)

	if totalA == 0 || totalB == 0 {
		return 0.0
	}

	// ASTScore = common_subtrees / min(total_subtrees_A, total_subtrees_B)
	minTotal := totalA
	if totalB < minTotal {
		minTotal = totalB
	}

	if minTotal == 0 {
		return 0.0
	}

	return float64(commonCount) / float64(minTotal)
}

// buildSubtreeHashes builds a multiset of subtree hashes using post-order traversal
func buildSubtreeHashes(node *models.ASTNode) map[string]bool {
	hashes := make(map[string]bool)
	buildSubtreeHashesRecursive(node, hashes)
	return hashes
}

// buildSubtreeHashesRecursive recursively builds subtree hashes (post-order)
func buildSubtreeHashesRecursive(node *models.ASTNode, hashes map[string]bool) {
	if node == nil {
		return
	}

	// Process children first (post-order)
	childHashes := make([]string, 0)
	if node.Children != nil {
		for _, child := range node.Children {
			buildSubtreeHashesRecursive(child, hashes)
			// Get hash of child (simplified - in real implementation, store hash in node)
			childHash := computeNodeHash(child)
			childHashes = append(childHashes, childHash)
		}
	}

	// Compute hash for this node: hash(node_type + child_hashes)
	sort.Strings(childHashes) // Sort for consistency
	hashInput := node.Type
	for _, ch := range childHashes {
		hashInput += ch
	}

	nodeHash := computeHash(hashInput)
	hashes[nodeHash] = true
}

// computeNodeHash computes hash for a node
func computeNodeHash(node *models.ASTNode) string {
	if node == nil {
		return ""
	}

	hashInput := node.Type
	if node.Name != "" {
		hashInput += node.Name
	}
	if node.ReturnType != "" {
		hashInput += node.ReturnType
	}

	// Include children hashes if any
	if node.Children != nil {
		childHashes := make([]string, 0)
		for _, child := range node.Children {
			childHashes = append(childHashes, computeNodeHash(child))
		}
		sort.Strings(childHashes)
		for _, ch := range childHashes {
			hashInput += ch
		}
	}

	return computeHash(hashInput)
}

// computeHash computes SHA256 hash of a string
func computeHash(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}
