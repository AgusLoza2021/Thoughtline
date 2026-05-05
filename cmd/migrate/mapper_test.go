package migrate

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// ---- MapType tests ----

func TestMapType(t *testing.T) {
	tests := []struct {
		name        string
		engramType  string
		wantType    memory.Type
		wantTag     string // empty = no origin-type tag expected
		wantErr     bool
	}{
		{
			name:       "bugfix maps 1:1 no tag",
			engramType: "bugfix",
			wantType:   memory.TypeBugfix,
			wantTag:    "",
		},
		{
			name:       "preference maps 1:1 no tag",
			engramType: "preference",
			wantType:   memory.TypePreference,
			wantTag:    "",
		},
		{
			name:       "decision maps 1:1 no tag",
			engramType: "decision",
			wantType:   memory.TypeDecision,
			wantTag:    "",
		},
		{
			name:       "architecture maps 1:1 no tag",
			engramType: "architecture",
			wantType:   memory.TypeArchitecture,
			wantTag:    "",
		},
		{
			name:       "pattern coerces to convention with origin tag",
			engramType: "pattern",
			wantType:   memory.TypeConvention,
			wantTag:    "origin-type:pattern",
		},
		{
			name:       "config coerces to convention with origin tag",
			engramType: "config",
			wantType:   memory.TypeConvention,
			wantTag:    "origin-type:config",
		},
		{
			name:       "discovery coerces to convention with origin tag",
			engramType: "discovery",
			wantType:   memory.TypeConvention,
			wantTag:    "origin-type:discovery",
		},
		{
			name:       "manual coerces to convention with origin tag",
			engramType: "manual",
			wantType:   memory.TypeConvention,
			wantTag:    "origin-type:manual",
		},
		{
			name:       "unknown type wizard coerces to convention with origin tag",
			engramType: "wizard",
			wantType:   memory.TypeConvention,
			wantTag:    "origin-type:wizard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotTags, err := MapType(tt.engramType)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MapType(%q) error = %v, wantErr = %v", tt.engramType, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if gotType != tt.wantType {
				t.Errorf("MapType(%q) type = %q, want %q", tt.engramType, gotType, tt.wantType)
			}
			if tt.wantTag == "" {
				if len(gotTags) != 0 {
					t.Errorf("MapType(%q) expected no tags, got %v", tt.engramType, gotTags)
				}
			} else {
				found := false
				for _, tag := range gotTags {
					if tag == tt.wantTag {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("MapType(%q) expected tag %q in %v", tt.engramType, tt.wantTag, gotTags)
				}
			}
		})
	}
}

// ---- MapTimestamp tests ----

func TestMapTimestamp(t *testing.T) {
	// 2024-03-15T10:30:00Z → unix-ms = 1710498600000
	// (verified: date -d "2024-03-15T10:30:00Z" +%s → 1710498600)
	const knownUTCUnixMS int64 = 1710498600000

	tests := []struct {
		name    string
		iso     string
		wantMS  int64
		wantErr bool
	}{
		{
			name:   "UTC Z suffix",
			iso:    "2024-03-15T10:30:00Z",
			wantMS: knownUTCUnixMS,
		},
		{
			name:   "explicit +00:00 offset same result",
			iso:    "2024-03-15T10:30:00+00:00",
			wantMS: knownUTCUnixMS,
		},
		{
			name:   "SQLite space-separated format treated as UTC",
			iso:    "2024-03-15 10:30:00",
			wantMS: knownUTCUnixMS,
		},
		{
			name:    "garbage string returns error",
			iso:     "not-a-date",
			wantErr: true,
		},
		{
			name:    "empty string returns error",
			iso:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MapTimestamp(tt.iso)
			if (err != nil) != tt.wantErr {
				t.Fatalf("MapTimestamp(%q) err = %v, wantErr = %v", tt.iso, err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got != tt.wantMS {
				t.Errorf("MapTimestamp(%q) = %d, want %d", tt.iso, got, tt.wantMS)
			}
		})
	}
}

// ---- MapRow tests ----

// nopLogger satisfies Logger without recording anything. Safe for mapper tests
// that don't need to inspect log output.
type nopLoggerT struct{}

func (nopLoggerT) Log(_ string, _ string, _ map[string]string) {}

func baseObservation() EngramObservation {
	return EngramObservation{
		SyncID:    "obs-abc123",
		Type:      "bugfix",
		Title:     "Fix crash on startup",
		Content:   "Root cause: nil pointer in init().\nFix: guard with early return.",
		Project:   "enchanted-inn",
		Scope:     "project",
		TopicKey:  "bug/startup-crash",
		CreatedAt: "2024-03-15T10:30:00Z",
		UpdatedAt: "2024-03-15T11:00:00Z",
	}
}

func TestMapRow(t *testing.T) {
	t.Run("happy path - all columns transformed", func(t *testing.T) {
		obs := baseObservation()
		m, err := MapRow(obs, nopLoggerT{})
		if err != nil {
			t.Fatalf("MapRow happy path error: %v", err)
		}
		if m.Type != memory.TypeBugfix {
			t.Errorf("Type = %q, want %q", m.Type, memory.TypeBugfix)
		}
		if m.Project != "enchanted-inn" {
			t.Errorf("Project = %q, want %q", m.Project, "enchanted-inn")
		}
		if m.TopicKey != "bug/startup-crash" {
			t.Errorf("TopicKey = %q, want %q", m.TopicKey, "bug/startup-crash")
		}
		if m.SessionID != "" {
			t.Errorf("SessionID should be cleared, got %q", m.SessionID)
		}
		if m.NormalizedHash != "" {
			t.Errorf("NormalizedHash should be left zero (storage recomputes), got %q", m.NormalizedHash)
		}
		if m.CreatedAt.IsZero() {
			t.Errorf("CreatedAt should be populated")
		}
		if m.UpdatedAt.IsZero() {
			t.Errorf("UpdatedAt should be populated")
		}
	})

	t.Run("preference scope forced to personal regardless of source scope", func(t *testing.T) {
		obs := baseObservation()
		obs.Type = "preference"
		obs.Scope = "project" // source says project; we must force personal
		m, err := MapRow(obs, nopLoggerT{})
		if err != nil {
			t.Fatalf("MapRow preference error: %v", err)
		}
		if m.Scope != memory.ScopePersonal {
			t.Errorf("preference scope = %q, want %q", m.Scope, memory.ScopePersonal)
		}
	})

	t.Run("title exactly 200 runes - no truncation, no log", func(t *testing.T) {
		obs := baseObservation()
		obs.Title = strings.Repeat("a", memory.MaxTitleChars)
		m, err := MapRow(obs, nopLoggerT{})
		if err != nil {
			t.Fatalf("200-rune title should not error: %v", err)
		}
		if utf8.RuneCountInString(m.Title) != memory.MaxTitleChars {
			t.Errorf("title rune count = %d, want %d", utf8.RuneCountInString(m.Title), memory.MaxTitleChars)
		}
	})

	t.Run("title 201 runes - truncated to 200, logger called with sync_id", func(t *testing.T) {
		type logEntry struct{ syncID string }
		var logged []logEntry
		spy := spyLogger{onLog: func(_ string, _ string, fields map[string]string) {
			if sid, ok := fields["sync_id"]; ok {
				logged = append(logged, logEntry{syncID: sid})
			}
		}}

		obs := baseObservation()
		obs.Title = strings.Repeat("b", memory.MaxTitleChars+1) // 201 runes
		m, err := MapRow(obs, spy)
		if err != nil {
			t.Fatalf("201-rune title should not error (truncate, not reject): %v", err)
		}
		if utf8.RuneCountInString(m.Title) != memory.MaxTitleChars {
			t.Errorf("truncated title rune count = %d, want %d", utf8.RuneCountInString(m.Title), memory.MaxTitleChars)
		}
		if len(logged) == 0 {
			t.Errorf("expected logger call with sync_id on truncation, got none")
		} else if logged[0].syncID != obs.SyncID {
			t.Errorf("logged sync_id = %q, want %q", logged[0].syncID, obs.SyncID)
		}
	})

	t.Run("content at MaxContentBytes - no error", func(t *testing.T) {
		obs := baseObservation()
		obs.Content = strings.Repeat("x", memory.MaxContentBytes)
		_, err := MapRow(obs, nopLoggerT{})
		if err != nil {
			t.Errorf("content at limit should not error: %v", err)
		}
	})

	t.Run("content MaxContentBytes+1 - MapRow returns error", func(t *testing.T) {
		obs := baseObservation()
		obs.Content = strings.Repeat("x", memory.MaxContentBytes+1)
		_, err := MapRow(obs, nopLoggerT{})
		if err == nil {
			t.Errorf("oversized content should return error")
		}
	})

	t.Run("valid topic_key copied verbatim", func(t *testing.T) {
		obs := baseObservation()
		obs.TopicKey = "architecture/storage-layer"
		m, err := MapRow(obs, nopLoggerT{})
		if err != nil {
			t.Fatalf("valid topic_key should not error: %v", err)
		}
		if m.TopicKey != "architecture/storage-layer" {
			t.Errorf("TopicKey = %q, want %q", m.TopicKey, "architecture/storage-layer")
		}
	})

	t.Run("invalid topic_key cleared silently", func(t *testing.T) {
		obs := baseObservation()
		obs.TopicKey = "Invalid/UPPER" // uppercase — invalid per Thoughtline regex
		m, err := MapRow(obs, nopLoggerT{})
		if err != nil {
			t.Fatalf("invalid topic_key should not error (silently cleared): %v", err)
		}
		if m.TopicKey != "" {
			t.Errorf("invalid topic_key should be cleared, got %q", m.TopicKey)
		}
	})

	t.Run("bogus scope returns error", func(t *testing.T) {
		obs := baseObservation()
		obs.Scope = "bogus"
		_, err := MapRow(obs, nopLoggerT{})
		if err == nil {
			t.Errorf("bogus scope should return error")
		}
	})
}

// spyLogger records calls; used in tests that need to inspect log output.
type spyLogger struct {
	onLog func(level, msg string, fields map[string]string)
}

func (s spyLogger) Log(level, msg string, fields map[string]string) {
	if s.onLog != nil {
		s.onLog(level, msg, fields)
	}
}
