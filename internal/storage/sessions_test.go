package storage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// ----------------------------- StartSession --------------------------------

func TestStartSession_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "claude-code")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if sess.ID == "" {
		t.Errorf("session id must be set")
	}
	parsed, err := uuid.Parse(sess.ID)
	if err != nil || parsed.Version() != 7 {
		t.Errorf("session id must be UUIDv7, got %q", sess.ID)
	}
	if sess.Project != "enchanted-inn" || sess.AgentLabel != "claude-code" {
		t.Errorf("metadata mismatch: %+v", sess)
	}
	if sess.StartedAt.IsZero() {
		t.Errorf("started_at must be populated")
	}
	if sess.EndedAt != nil {
		t.Errorf("freshly started session must be open")
	}
	if sess.Summary != "" {
		t.Errorf("summary must be empty until tl_session_summary, got %q", sess.Summary)
	}
}

func TestStartSession_RejectsEmptyProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	_, err := st.StartSession(ctx, "", "claude-code")
	if !errors.Is(err, memory.ErrEmptySessionProject) {
		t.Errorf("expected ErrEmptySessionProject, got %v", err)
	}
}

func TestStartSession_RejectsAgentLabelTooLong(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	long := strings.Repeat("x", memory.MaxAgentLabelChars+1)
	_, err := st.StartSession(ctx, "enchanted-inn", long)
	if !errors.Is(err, memory.ErrAgentLabelTooLong) {
		t.Errorf("expected ErrAgentLabelTooLong, got %v", err)
	}
}

func TestStartSession_AgentLabelOptional(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("agent label should be optional: %v", err)
	}
	if sess.AgentLabel != "" {
		t.Errorf("expected empty label, got %q", sess.AgentLabel)
	}
}

// ----------------------------- GetSession ----------------------------------

func TestGetSession_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	created, err := st.StartSession(ctx, "enchanted-inn", "claude-code")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	got, err := st.GetSession(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != created.ID || got.Project != created.Project || got.AgentLabel != created.AgentLabel {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, created)
	}
	if !got.StartedAt.Equal(created.StartedAt) {
		t.Errorf("started_at mismatch: got %v want %v", got.StartedAt, created.StartedAt)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	id := uuid.Must(uuid.NewV7()).String()
	_, err := st.GetSession(ctx, id)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

// ----------------------------- EndSession ----------------------------------

func TestEndSession_HappyPath(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 4, 30, 14, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })

	sess, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	st.SetClock(func() time.Time { return t0.Add(45 * time.Minute) })

	ended, err := st.EndSession(ctx, sess.ID, "Worked on lantern bake.")
	if err != nil {
		t.Fatalf("end: %v", err)
	}
	if ended.EndedAt == nil {
		t.Fatalf("ended_at must be populated")
	}
	if ended.Summary != "Worked on lantern bake." {
		t.Errorf("summary mismatch: %q", ended.Summary)
	}
	if !ended.EndedAt.After(ended.StartedAt) {
		t.Errorf("ended_at must be after started_at")
	}
	if got := ended.Duration(); got != 45*time.Minute {
		t.Errorf("duration = %v, want 45m", got)
	}
}

func TestEndSession_NotFound(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	id := uuid.Must(uuid.NewV7()).String()
	_, err := st.EndSession(ctx, id, "summary")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestEndSession_AlreadyEndedRejected(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := st.EndSession(ctx, sess.ID, "first close"); err != nil {
		t.Fatalf("first end: %v", err)
	}

	_, err = st.EndSession(ctx, sess.ID, "second close")
	if !errors.Is(err, ErrSessionAlreadyEnded) {
		t.Errorf("expected ErrSessionAlreadyEnded, got %v", err)
	}
}

func TestEndSession_RejectsOversizedSummary(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	tooLong := strings.Repeat("x", memory.MaxSessionSummaryBytes+1)
	_, err = st.EndSession(ctx, sess.ID, tooLong)
	if !errors.Is(err, memory.ErrSessionSummaryTooLong) {
		t.Errorf("expected ErrSessionSummaryTooLong, got %v", err)
	}
}

// --------------------------- RecentSessions --------------------------------

func TestRecentSessions_OrdersByStartedAtDesc(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	t0 := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	st.SetClock(func() time.Time { return t0 })
	first, err := st.StartSession(ctx, "enchanted-inn", "claude-code")
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	st.SetClock(func() time.Time { return t0.Add(time.Hour) })
	second, err := st.StartSession(ctx, "enchanted-inn", "cursor")
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	results, err := st.RecentSessions(ctx, "enchanted-inn", 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(results))
	}
	if results[0].ID != second.ID || results[1].ID != first.ID {
		t.Errorf("expected newer session first, got %v / %v", results[0].ID, results[1].ID)
	}
}

func TestRecentSessions_FiltersByProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	if _, err := st.StartSession(ctx, "alpha", ""); err != nil {
		t.Fatalf("alpha: %v", err)
	}
	if _, err := st.StartSession(ctx, "beta", ""); err != nil {
		t.Fatalf("beta: %v", err)
	}

	results, err := st.RecentSessions(ctx, "alpha", 0)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(results) != 1 || results[0].Project != "alpha" {
		t.Errorf("project filter broken, got %+v", results)
	}
}

func TestRecentSessions_RequiresProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	if _, err := st.RecentSessions(ctx, "", 0); err == nil {
		t.Errorf("empty project must be rejected")
	}
}

func TestRecentSessions_LimitDefaultAndClamp(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 60; i++ {
		if _, err := st.StartSession(ctx, "enchanted-inn", ""); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	def, err := st.RecentSessions(ctx, "enchanted-inn", 0)
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if len(def) != DefaultSearchLimit {
		t.Errorf("default limit must be %d, got %d", DefaultSearchLimit, len(def))
	}

	clamped, err := st.RecentSessions(ctx, "enchanted-inn", 999)
	if err != nil {
		t.Fatalf("clamped: %v", err)
	}
	if len(clamped) != MaxSearchLimit {
		t.Errorf("clamped limit must be %d, got %d", MaxSearchLimit, len(clamped))
	}
}

// ----------------- Save x Session: persistence + sticky -------------------

func TestSave_PersistsSessionID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	m := sampleMemory()
	m.SessionID = sess.ID
	saved, _, err := st.Save(ctx, m)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.SessionID != sess.ID {
		t.Errorf("session_id round-trip mismatch: got %q want %q", saved.SessionID, sess.ID)
	}

	got, err := st.GetByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SessionID != sess.ID {
		t.Errorf("session_id not persisted, got %q", got.SessionID)
	}
}

// REGRESSION GUARD for the upsert-clobber-session bug we caught in audit.
// Once a memory is associated to a session, a subsequent tl_save with the
// same topic_key but no session_id MUST preserve the prior session_id
// (COALESCE behaviour). Otherwise we'd silently lose session linkage.
func TestSave_UpsertDoesNotClobberSessionID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sess, err := st.StartSession(ctx, "enchanted-inn", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	first := sampleMemory()
	first.TopicKey = "scene/lantern"
	first.SessionID = sess.ID
	if _, _, err := st.Save(ctx, first); err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Re-save with same topic_key, content changed, NO session_id provided.
	second := first
	second.SessionID = ""
	second.Content = first.Content + "\nupdated body"
	updated, action, err := st.Save(ctx, second)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if action != ActionUpdated {
		t.Fatalf("expected ActionUpdated, got %s", action)
	}
	if updated.SessionID != sess.ID {
		t.Errorf("session_id was clobbered by upsert; got %q, want %q", updated.SessionID, sess.ID)
	}

	// Confirm via fresh read.
	got, err := st.GetByID(ctx, updated.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.SessionID != sess.ID {
		t.Errorf("session_id not preserved across upsert in DB, got %q", got.SessionID)
	}
}

// REGRESSION GUARD: cross-project session attachment must be rejected.
// A memory in project=A cannot reference a session in project=B.
func TestSave_RejectsCrossProjectSession(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sessAlpha, err := st.StartSession(ctx, "alpha", "")
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	m := sampleMemory()
	m.Project = "beta"
	m.SessionID = sessAlpha.ID

	_, _, err = st.Save(ctx, m)
	if !errors.Is(err, ErrSessionProjectMismatch) {
		t.Errorf("expected ErrSessionProjectMismatch, got %v", err)
	}
}

func TestSave_RejectsUnknownSessionID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	bogus := uuid.Must(uuid.NewV7()).String()
	m := sampleMemory()
	m.SessionID = bogus

	_, _, err := st.Save(ctx, m)
	if !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("expected ErrSessionNotFound for unknown session, got %v", err)
	}
}

// ----------------- Schema migration regression -----------------------------

// Fresh DB must have session_id column on memories AND idx_memories_session.
// This pins the v2 migration even on first-open.
func TestMigration_FreshDBHasSessionColumn(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	has, err := columnExists(ctx, st.db, "memories", "session_id")
	if err != nil {
		t.Fatalf("column check: %v", err)
	}
	if !has {
		t.Errorf("fresh DB must have memories.session_id column after migration")
	}
}

// Re-opening an existing v2 DB must NOT fail (idempotent ALTER TABLE check).
func TestMigration_IsIdempotentOnReopen(t *testing.T) {
	st := newTestStorage(t)
	if err := st.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	// newTestStorage uses t.TempDir; reopen the same path.
	// Note: newTestStorage doesn't expose its path. Instead, reopen via Open
	// on an in-memory DB simulation: just verify the helper doesn't error
	// when the column already exists by calling migrate() twice on the same
	// connection.
	st2 := newTestStorage(t)
	ctx := context.Background()
	// Run migrate again — must not error.
	if err := st2.migrate(ctx); err != nil {
		t.Errorf("re-running migrate must be idempotent, got %v", err)
	}
}
