package plagiarism

import (
	"math"

	"github.com/RishiKendai/aegis/internal/models"
)

// CFGSimilarity calculates similarity using CFG feature vector distance
func CFGSimilarity(artifactA, artifactB *models.Artifact) float64 {
	if artifactA.CFG == nil || artifactB.CFG == nil {
		return 0.0
	}

	// Extract features
	featuresA := extractCFGFeatures(artifactA.CFG)
	featuresB := extractCFGFeatures(artifactB.CFG)

	// Calculate normalized distance
	distance := euclideanDistance(featuresA, featuresB)
	maxDistance := calculateMaxDistance(featuresA, featuresB)

	if maxDistance == 0 {
		return 1.0 // Identical CFGs
	}

	// CFGScore = 1 - (||vecA - vecB|| / max_distance)
	score := 1.0 - (distance / maxDistance)
	if score < 0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// extractCFGFeatures extracts 6-dimensional feature vector:
// [#Nodes, #Edges, #Branches, #Loops, Max depth, Cyclomatic complexity]
func extractCFGFeatures(cfg *models.CFG) [6]float64 {
	features := [6]float64{}

	// #Nodes
	features[0] = float64(len(cfg.Nodes))

	// #Edges
	features[1] = float64(len(cfg.Edges))

	// #Branches (edges with type "BRANCH" or conditional edges)
	branches := 0
	for _, edge := range cfg.Edges {
		if edge.Type == "BRANCH" || edge.Type == "CONDITIONAL" {
			branches++
		}
	}
	features[2] = float64(branches)

	// #Loops (detect cycles in CFG)
	features[3] = float64(detectLoops(cfg))

	// Max depth (longest path from entry to exit)
	features[4] = float64(calculateMaxDepth(cfg))

	// Cyclomatic complexity = E - N + 2P
	// E = edges, N = nodes, P = connected components (usually 1)
	features[5] = features[1] - features[0] + 2.0

	return features
}

// detectLoops detects cycles in the CFG
func detectLoops(cfg *models.CFG) int {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, edge := range cfg.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	// Simple cycle detection - count back edges
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	loopCount := 0

	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				dfs(neighbor)
			} else if recStack[neighbor] {
				loopCount++ // Back edge found
			}
		}

		recStack[node] = false
	}

	// Start DFS from entry node
	for _, node := range cfg.Nodes {
		if node.Type == "ENTRY" && !visited[node.ID] {
			dfs(node.ID)
		}
	}

	return loopCount
}

// calculateMaxDepth calculates the longest path from entry to exit
func calculateMaxDepth(cfg *models.CFG) int {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, edge := range cfg.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	// Find entry node
	var entryID string
	for _, node := range cfg.Nodes {
		if node.Type == "ENTRY" {
			entryID = node.ID
			break
		}
	}

	if entryID == "" {
		return 0
	}

	// BFS to find max depth
	maxDepth := 0
	queue := []struct {
		id    string
		depth int
	}{{entryID, 0}}
	visited := make(map[string]bool)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if visited[curr.id] {
			continue
		}
		visited[curr.id] = true

		if curr.depth > maxDepth {
			maxDepth = curr.depth
		}

		for _, neighbor := range adj[curr.id] {
			if !visited[neighbor] {
				queue = append(queue, struct {
					id    string
					depth int
				}{neighbor, curr.depth + 1})
			}
		}
	}

	return maxDepth
}

// euclideanDistance calculates Euclidean distance between two feature vectors
func euclideanDistance(vecA, vecB [6]float64) float64 {
	sum := 0.0
	for i := 0; i < 6; i++ {
		diff := vecA[i] - vecB[i]
		sum += diff * diff
	}
	return math.Sqrt(sum)
}

// calculateMaxDistance calculates the maximum possible distance
func calculateMaxDistance(vecA, vecB [6]float64) float64 {
	// Use sum of magnitudes as approximation
	sumA := 0.0
	sumB := 0.0
	for i := 0; i < 6; i++ {
		sumA += vecA[i] * vecA[i]
		sumB += vecB[i] * vecB[i]
	}
	return math.Sqrt(sumA) + math.Sqrt(sumB)
}
