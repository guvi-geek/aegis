package models

// Submission represents a submission from Redis stream
type Submission struct {
	AttemptID  string `json:"attemptID"`
	SourceCode string `json:"sourceCode"`
	Language   string `json:"language"`
	LangCode   string `json:"langCode"`
	Email      string `json:"email"`
	TestID     string `json:"testId"`
	DriveID    string `json:"driveId"`
	QID        int64  `json:"qId"`
	Difficulty string `json:"difficulty"`
}
