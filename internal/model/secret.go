package model

import "time"

// SecretMetadata is stored in DynamoDB. Values live in AWS Secrets Manager.
type SecretMetadata struct {
	Name         string    `json:"name" dynamodbav:"name"`
	Owners       []string  `json:"owners" dynamodbav:"owners"`
	Retrievers   []string  `json:"retrievers" dynamodbav:"retrievers"`
	ASMSecretARN string    `json:"-" dynamodbav:"asm_secret_arn"`
	LastUpdated  time.Time `json:"last_updated" dynamodbav:"last_updated"`
}

// SecretValues are the green/blue blobs stored in ASM.
type SecretValues struct {
	Value1 string `json:"value1,omitempty"`
	Value2 string `json:"value2,omitempty"`
}

// Secret is the full secret returned on retrieve.
type Secret struct {
	Name        string    `json:"name"`
	Owners      []string  `json:"owners"`
	Retrievers  []string  `json:"retrievers"`
	Value1      string    `json:"value1,omitempty"`
	Value2      string    `json:"value2,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}

// SecretSummary is returned when listing secrets (no values).
type SecretSummary struct {
	Name        string    `json:"name"`
	Owners      []string  `json:"owners"`
	Retrievers  []string  `json:"retrievers"`
	LastUpdated time.Time `json:"last_updated"`
	CanUpdate   bool      `json:"can_update"`
}

// CreateSecretRequest is the body for POST /secrets.
type CreateSecretRequest struct {
	Name       string   `json:"name"`
	Owners     []string `json:"owners"`
	Retrievers []string `json:"retrievers"`
	Value1     string   `json:"value1,omitempty"`
	Value2     string   `json:"value2,omitempty"`
}

// UpdateSecretRequest is the body for PUT /secrets/{name}.
type UpdateSecretRequest struct {
	Owners     []string `json:"owners,omitempty"`
	Retrievers []string `json:"retrievers,omitempty"`
	Value1     *string  `json:"value1,omitempty"`
	Value2     *string  `json:"value2,omitempty"`
}

// PollEntry is one item in a poll request.
type PollEntry struct {
	Name        string    `json:"name"`
	LastUpdated time.Time `json:"last_updated"`
}

// PollRequest is the body for POST /secrets/poll.
type PollRequest struct {
	Secrets []PollEntry `json:"secrets"`
}

// PollResponse lists secrets that changed since the client's last known state.
type PollResponse struct {
	Updated []SecretSummary `json:"updated"`
}
