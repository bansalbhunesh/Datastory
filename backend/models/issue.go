package models

// FailedTestSummary captures enough detail for RCA + incident copy.
type FailedTestSummary struct {
	Name        string `json:"name"`
	FQN         string `json:"fqn,omitempty"`
	Status      string `json:"status"`
	Result      string `json:"result,omitempty"`
	UpdatedAt   int64  `json:"updatedAt,omitempty"`
	Description string `json:"description,omitempty"`
}
