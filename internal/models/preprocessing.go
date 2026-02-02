package models

// PreprocessingResponse represents the response from Astra preprocessing API
type PreprocessingResponse struct {
	EmailID       string            `json:"email"`
	AttemptID     string            `json:"attemptId"`
	DriveID       string            `json:"testId"`
	Language      string            `json:"language"`
	Preprocessing PreprocessingData `json:"preprocessing"`
}

// PreprocessingData contains the preprocessing results
type PreprocessingData struct {
	Tokens           []string      `json:"tokens"`
	NormalizedTokens []string      `json:"normalizedTokens"`
	AST              *ASTNode      `json:"ast"`
	CFG              *CFG          `json:"cfg"`
	Fingerprints     *Fingerprints `json:"fingerprints"`
}

// ASTNode represents an AST node
type ASTNode struct {
	Type       string                   `json:"type"`
	Name       string                   `json:"name,omitempty"`
	Modifiers  []string                 `json:"modifiers,omitempty"`
	ReturnType string                   `json:"returnType,omitempty"`
	Parameters []*Parameter             `json:"parameters,omitempty"`
	Body       map[string]interface{}   `json:"body,omitempty"`
	Children   []*ASTNode               `json:"children,omitempty"`
	Expression map[string]interface{}   `json:"expression,omitempty"`
	Operator   string                   `json:"operator,omitempty"`
	Left       map[string]interface{}   `json:"left,omitempty"`
	Right      map[string]interface{}   `json:"right,omitempty"`
	Statements []map[string]interface{} `json:"statements,omitempty"`
}

// Parameter represents a function parameter
type Parameter struct {
	Type      string `json:"type"`
	ParamType string `json:"paramType"`
	Name      string `json:"name"`
}

// CFG represents a Control Flow Graph
type CFG struct {
	Nodes []*CFGNode `json:"nodes"`
	Edges []*CFGEdge `json:"edges"`
}

// CFGNode represents a CFG node
type CFGNode struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Label      string `json:"label"`
	LineNumber int    `json:"lineNumber,omitempty"`
}

// CFGEdge represents a CFG edge
type CFGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

// Fingerprints represents fingerprint data
type Fingerprints struct {
	Method     string      `json:"method"`
	KGramSize  int         `json:"kGramSize"`
	WindowSize int         `json:"windowSize"`
	Hashes     []HashEntry `json:"hashes"`
}

// HashEntry represents a single hash entry
type HashEntry struct {
	Hash     string `json:"hash"`
	Position int    `json:"position"`
}

// PreprocessingError represents an error response from Astra API
type PreprocessingError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}
