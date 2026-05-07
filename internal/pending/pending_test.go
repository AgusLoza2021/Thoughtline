package pending_test

import (
	"errors"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/pending"
)

func validEvent() pending.Event {
	return pending.Event{
		SyncID:     "sync-abc",
		Project:    "enchanted-inn",
		SessionID:  "sess-1",
		EventType:  "PreToolUse",
		ToolName:   "Read",
		ToolUseID:  "tool-1",
		Payload:    `{"hook_event_name":"PreToolUse","session_id":"sess-1"}`,
		Hash:       "deadbeef",
		Status:     pending.StatusPending,
		CreatedAt:  time.Now(),
		CapturedAt: time.Now(),
	}
}

func TestValidate_AcceptsAllSixKnownEventTypes(t *testing.T) {
	known := []string{
		"SessionStart",
		"UserPromptSubmit",
		"PreToolUse",
		"PostToolUse",
		"Stop",
		"SessionEnd",
	}
	for _, et := range known {
		e := validEvent()
		e.EventType = et
		if err := pending.Validate(e); err != nil {
			t.Errorf("Validate rejected known event type %q: %v", et, err)
		}
	}
}

func TestValidate_RejectsEmptyEventType(t *testing.T) {
	e := validEvent()
	e.EventType = ""
	err := pending.Validate(e)
	if !errors.Is(err, pending.ErrEmptyEventType) {
		t.Errorf("expected ErrEmptyEventType, got %v", err)
	}
}

func TestValidate_RejectsUnknownEventType(t *testing.T) {
	e := validEvent()
	e.EventType = "UnknownEvent"
	err := pending.Validate(e)
	if !errors.Is(err, pending.ErrUnknownEventType) {
		t.Errorf("expected ErrUnknownEventType for unknown type, got %v", err)
	}
}

func TestValidate_RejectsEmptyProject(t *testing.T) {
	e := validEvent()
	e.Project = ""
	err := pending.Validate(e)
	if !errors.Is(err, pending.ErrEmptyProject) {
		t.Errorf("expected ErrEmptyProject, got %v", err)
	}
}

func TestValidate_RejectsEmptyPayload(t *testing.T) {
	e := validEvent()
	e.Payload = ""
	err := pending.Validate(e)
	if !errors.Is(err, pending.ErrEmptyPayload) {
		t.Errorf("expected ErrEmptyPayload, got %v", err)
	}
}

func TestValidate_RejectsInvalidStatus(t *testing.T) {
	e := validEvent()
	e.Status = "flying"
	err := pending.Validate(e)
	if !errors.Is(err, pending.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestValidate_AcceptsAllStatuses(t *testing.T) {
	statuses := []pending.Status{
		pending.StatusPending,
		pending.StatusPromoted,
		pending.StatusArchived,
	}
	for _, s := range statuses {
		e := validEvent()
		e.Status = s
		if err := pending.Validate(e); err != nil {
			t.Errorf("Validate rejected status %q: %v", s, err)
		}
	}
}
