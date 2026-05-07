package pending

import (
	"errors"
	"time"
)

// Status is the lifecycle state of a pending event.
type Status string

const (
	StatusPending  Status = "pending"
	StatusPromoted Status = "promoted"
	StatusArchived Status = "archived"
)

// KnownEventTypes is the exhaustive set of Claude Code hook event names that
// thoughtline hook accepts. Events outside this set are silently skipped
// (exit 0, log to stderr).
var KnownEventTypes = map[string]bool{
	"SessionStart":      true,
	"UserPromptSubmit":  true,
	"PreToolUse":        true,
	"PostToolUse":       true,
	"Stop":              true,
	"SessionEnd":        true,
}

// Event is the domain representation of a row in pending_events.
type Event struct {
	ID                int64
	SyncID            string
	Project           string
	SessionID         string  // "" when absent
	EventType         string
	ToolName          string  // "" when absent
	ToolUseID         string  // "" when absent
	Payload           string  // raw JSON as received on stdin
	Hash              string
	Status            Status
	PromotedMemoryID  int64   // 0 when not promoted
	PromotedAt        time.Time
	ArchivedAt        time.Time
	CreatedAt         time.Time
	CapturedAt        time.Time
}

// Sentinel errors for Validate.
var (
	ErrEmptyEventType   = errors.New("pending: event_type must not be empty")
	ErrUnknownEventType = errors.New("pending: unknown event_type (not one of the 6 accepted names)")
	ErrEmptyProject     = errors.New("pending: project must not be empty")
	ErrEmptyPayload     = errors.New("pending: payload must not be empty")
	ErrInvalidStatus    = errors.New("pending: status must be pending, promoted, or archived")
)

// Validate checks domain invariants for an Event. Storage callers should pass
// the event through Validate before inserting.
func Validate(e Event) error {
	if e.EventType == "" {
		return ErrEmptyEventType
	}
	if !KnownEventTypes[e.EventType] {
		return ErrUnknownEventType
	}
	if e.Project == "" {
		return ErrEmptyProject
	}
	if e.Payload == "" {
		return ErrEmptyPayload
	}
	switch e.Status {
	case StatusPending, StatusPromoted, StatusArchived:
		// valid
	default:
		return ErrInvalidStatus
	}
	return nil
}
