package memory

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Session bookends a coding interaction so the AI has a stable narrative
// across context compactions. Sessions are append-once: started by
// tl_session_start, closed exactly once by tl_session_summary. A Memory may
// optionally reference its parent session via Memory.SessionID.
//
// IDs are UUIDv7 strings — sortable by time, generated client-side.
type Session struct {
	// ID is a UUIDv7 string assigned by storage at start time.
	ID string
	// Project the session belongs to. Required.
	Project string
	// AgentLabel is an optional client-provided tag like "claude-code",
	// "cursor", "zed". Useful for cross-session forensics.
	AgentLabel string
	// StartedAt is when tl_session_start was called.
	StartedAt time.Time
	// EndedAt is non-nil once tl_session_summary has closed the session.
	EndedAt *time.Time
	// Summary is the structured end-of-session digest. Empty until ended.
	Summary string
}

// IsOpen reports whether the session has not yet been closed by
// tl_session_summary.
func (s Session) IsOpen() bool {
	return s.EndedAt == nil
}

// Duration returns ended_at - started_at. Zero if the session is still open.
func (s Session) Duration() time.Duration {
	if s.EndedAt == nil {
		return 0
	}
	return s.EndedAt.Sub(s.StartedAt)
}

// Limits exported so callers can show meaningful validation messages.
const (
	MaxAgentLabelChars     = 64
	MaxSessionSummaryBytes = MaxContentBytes
)

// Errors returned by ValidateSession. Tests assert against these directly.
var (
	ErrInvalidSessionID         = errors.New("session: invalid id (must be a UUIDv7)")
	ErrEmptySessionProject      = errors.New("session: project must not be empty")
	ErrAgentLabelTooLong        = errors.New("session: agent_label exceeds maximum characters")
	ErrEmptySessionSummary      = errors.New("session: summary must not be empty when ending a session")
	ErrSessionSummaryTooLong    = errors.New("session: summary exceeds maximum bytes")
	ErrSessionEndedBeforeStart  = errors.New("session: ended_at must be at or after started_at")
)

// ValidateSession enforces the Session invariants. Used by storage at
// StartSession / EndSession boundaries — the server layer never persists
// a Session that has not passed this check.
func ValidateSession(s Session) error {
	if s.ID != "" {
		// Empty ID is allowed at "about to start" call sites; storage assigns
		// the UUIDv7 before persisting. When non-empty, must be a valid
		// UUIDv7 (version byte = 7).
		parsed, err := uuid.Parse(s.ID)
		if err != nil {
			return ErrInvalidSessionID
		}
		if parsed.Version() != 7 {
			return ErrInvalidSessionID
		}
	}

	if s.Project == "" {
		return ErrEmptySessionProject
	}

	if len(s.AgentLabel) > MaxAgentLabelChars {
		return ErrAgentLabelTooLong
	}

	if len(s.Summary) > MaxSessionSummaryBytes {
		return ErrSessionSummaryTooLong
	}

	if s.EndedAt != nil && s.EndedAt.Before(s.StartedAt) {
		return ErrSessionEndedBeforeStart
	}

	return nil
}
