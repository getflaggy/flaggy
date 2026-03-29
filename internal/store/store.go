package store

import "github.com/getflaggy/flaggy/internal/models"

// Store defines the persistence interface for flags and rules.
type Store interface {
	// Flags
	CreateFlag(flag *models.Flag) error
	GetFlag(key string) (*models.Flag, error)
	ListFlags() ([]models.Flag, error)
	UpdateFlag(key string, req *models.UpdateFlagRequest) (*models.Flag, error)
	DeleteFlag(key string) error
	ToggleFlag(key string) (*models.Flag, error)

	// Rules
	CreateRule(flagKey string, rule *models.Rule) error
	UpdateRule(flagKey string, ruleID int64, req *models.CreateRuleRequest) (*models.Rule, error)
	DeleteRule(flagKey string, ruleID int64) error

	// Segments
	CreateSegment(segment *models.Segment) error
	GetSegment(key string) (*models.Segment, error)
	ListSegments() ([]models.Segment, error)
	UpdateSegment(key string, req *models.UpdateSegmentRequest) (*models.Segment, error)
	DeleteSegment(key string) error

	// Evaluation
	GetFlagForEvaluation(key string) (*models.Flag, error)

	// API Keys
	CreateAPIKey(key *models.APIKey, hashedKey string) error
	ListAPIKeys() ([]models.APIKey, error)
	ValidateAPIKey(hashedKey string) (*models.APIKey, error)
	RevokeAPIKey(id string) error

	Close() error
}
