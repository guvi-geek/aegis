package models

import (
	"time"
)

type Step string

const (
	StepIdle          Step = "idle"
	StepInitiated     Step = "initiated"
	StepStarted       Step = "started"
	StepPreprocessing Step = "preprocessing"
	StepFiltering     Step = "filtering"
	StepDeepAnalysis  Step = "deep_analysis"
	StepCompleted     Step = "completed"
)

// Artifact represents a plagiarism artifact stored in MongoDB
type Artifact struct {
	Email            string        `bson:"email" json:"email"`
	AttemptID        string        `bson:"attemptID" json:"attemptID"`
	TestID           string        `bson:"testId" json:"testId"`
	DriveID          string        `bson:"driveId" json:"driveId"`
	Difficulty       string        `bson:"difficulty" json:"difficulty"`
	SourceCode       string        `bson:"sourceCode" json:"sourceCode"`
	QID              int64         `bson:"qId" json:"qId"`
	Language         string        `bson:"language" json:"language"`
	LangCode         string        `bson:"langCode" json:"langCode"`
	Tokens           []string      `bson:"tokens" json:"tokens"`
	NormalizedTokens []string      `bson:"normalizedTokens" json:"normalizedTokens"`
	AST              *ASTNode      `bson:"ast" json:"ast"`
	CFG              *CFG          `bson:"cfg" json:"cfg"`
	Fingerprints     *Fingerprints `bson:"fingerprints" json:"fingerprints"`
	CreatedAt        time.Time     `bson:"createdAt" json:"createdAt"`
}

// CandidateResult represents a candidate's plagiarism result
type CandidateResult struct {
	Email            string              `bson:"email" json:"email"`
	AttemptID        string              `bson:"attemptID" json:"attemptID"`
	DriveID          string              `bson:"driveId" json:"driveId"`
	Risk             string              `bson:"risk" json:"risk"` // safe, suspicious, highly_suspicious, near_copy
	FlaggedQuestions []string            `bson:"flagged_qns" json:"flagged_qns"`
	PlagiarismPeers  map[string][]string `bson:"plagiarism_peers" json:"plagiarism_peers"` // qId -> []attemptId
	CodeSimilarity   int                 `bson:"code_similarity" json:"code_similarity"`
	AlgoSimilarity   int                 `bson:"algo_similarity" json:"algo_similarity"`
	PlagiarismStatus string              `bson:"plagiarism_status" json:"plagiarism_status"` // pending, completed, failed
	CreatedAt        time.Time           `bson:"createdAt" json:"createdAt"`
}

// TestReport represents an overall test plagiarism report
type TestReport struct {
	DriveID           string    `bson:"driveId" json:"driveId"`
	Risk              string    `bson:"risk" json:"risk"`     // safe, moderate, high, critical
	Status            string    `bson:"status" json:"status"` // pending, completed, failed
	CreatedAt         time.Time `bson:"createdAt" json:"createdAt"`
	FlaggedQuestions  []string  `bson:"flagged_qns" json:"flagged_qns"`
	FlaggedCandidates int       `bson:"flagged_candidates" json:"flagged_candidates"`
	TotalAnalyzed     int       `bson:"total_analyzed" json:"total_analyzed"`
}

// ComputeRequest represents a request to compute plagiarism
type ComputeRequest struct {
	DriveID string `json:"driveId" binding:"required"`
}

// ComputeResponse represents the response from compute endpoint
type ComputeResponse struct {
	Step   Step   `json:"step"`
	TestID string `json:"testId"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}
