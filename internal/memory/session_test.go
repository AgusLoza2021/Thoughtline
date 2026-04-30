package memory

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newV7ID(t *testing.T) string {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid v7: %v", err)
	}
	return id.String()
}

func validSession(t *testing.T) Session {
	t.Helper()
	return Session{
		ID:        newV7ID(t),
		Project:   "enchanted-inn",
		StartedAt: time.Date(2026, 4, 30, 14, 0, 0, 0, time.UTC),
	}
}

func TestValidateSession_HappyPath(t *testing.T) {
	if err := ValidateSession(validSession(t)); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateSession_EmptyIDIsAllowed(t *testing.T) {
	// Storage assigns the ID — callers may pre-validate with an empty ID.
	s := validSession(t)
	s.ID = ""
	if err := ValidateSession(s); err != nil {
		t.Errorf("empty id should be allowed pre-storage, got %v", err)
	}
}

func TestValidateSession_InvalidUUID(t *testing.T) {
	s := validSession(t)
	s.ID = "not-a-uuid"
	if err := ValidateSession(s); !errors.Is(err, ErrInvalidSessionID) {
		t.Errorf("expected ErrInvalidSessionID, got %v", err)
	}
}

func TestValidateSession_RejectsNonV7UUID(t *testing.T) {
	s := validSession(t)
	// A v4 UUID must be rejected — we lock the format to v7 for sortability.
	s.ID = uuid.New().String() // v4 by default
	if err := ValidateSession(s); !errors.Is(err, ErrInvalidSessionID) {
		t.Errorf("v4 UUID must be rejected, got %v", err)
	}
}

func TestValidateSession_EmptyProject(t *testing.T) {
	s := validSession(t)
	s.Project = ""
	if err := ValidateSession(s); !errors.Is(err, ErrEmptySessionProject) {
		t.Errorf("expected ErrEmptySessionProject, got %v", err)
	}
}

func TestValidateSession_AgentLabelTooLong(t *testing.T) {
	s := validSession(t)
	s.AgentLabel = strings.Repeat("x", MaxAgentLabelChars+1)
	if err := ValidateSession(s); !errors.Is(err, ErrAgentLabelTooLong) {
		t.Errorf("expected ErrAgentLabelTooLong, got %v", err)
	}
}

func TestValidateSession_AgentLabelAtLimit(t *testing.T) {
	s := validSession(t)
	s.AgentLabel = strings.Repeat("x", MaxAgentLabelChars)
	if err := ValidateSession(s); err != nil {
		t.Errorf("label at exact limit should pass, got %v", err)
	}
}

func TestValidateSession_SummaryTooLong(t *testing.T) {
	s := validSession(t)
	s.Summary = strings.Repeat("x", MaxSessionSummaryBytes+1)
	if err := ValidateSession(s); !errors.Is(err, ErrSessionSummaryTooLong) {
		t.Errorf("expected ErrSessionSummaryTooLong, got %v", err)
	}
}

func TestValidateSession_EndedBeforeStarted(t *testing.T) {
	s := validSession(t)
	end := s.StartedAt.Add(-time.Second)
	s.EndedAt = &end
	if err := ValidateSession(s); !errors.Is(err, ErrSessionEndedBeforeStart) {
		t.Errorf("expected ErrSessionEndedBeforeStart, got %v", err)
	}
}

func TestSession_IsOpenAndDuration(t *testing.T) {
	s := validSession(t)
	if !s.IsOpen() {
		t.Errorf("a session with EndedAt=nil must be open")
	}
	if s.Duration() != 0 {
		t.Errorf("open session duration must be zero, got %v", s.Duration())
	}

	end := s.StartedAt.Add(45 * time.Minute)
	s.EndedAt = &end
	if s.IsOpen() {
		t.Errorf("a session with EndedAt set must be closed")
	}
	if got := s.Duration(); got != 45*time.Minute {
		t.Errorf("duration = %v, want 45m", got)
	}
}
