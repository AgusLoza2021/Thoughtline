package storage

// A2 — DedupeCheck unit tests (REQ-CAPS-A2-001 through REQ-CAPS-A2-005).
//
// Each test seeds the database directly via Save/SoftDelete, then calls
// DedupeCheck and asserts the returned (found, existingID, existingCreatedAt).
// Clock injection (SetClock) is used to place rows at deterministic timestamps
// so window arithmetic is verifiable without real-time sleeps.

import (
	"context"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// t0 is a fixed base time used across all A2 tests to make assertions readable.
var t0 = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// seedMemoryAt saves a non-topic-keyed memory at the given absolute time and
// returns the persisted Memory. The storage clock is restored after the insert
// so callers can set their own clock for the DedupeCheck call.
func seedMemoryAt(t *testing.T, st *Storage, brainID int64, m memory.Memory, at time.Time) memory.Memory {
	t.Helper()
	st.SetClock(func() time.Time { return at })
	saved, _, err := st.Save(context.Background(), brainID, m)
	if err != nil {
		t.Fatalf("seedMemoryAt: %v", err)
	}
	return saved
}

// TestDedupeCheck_HitWithinWindow_ReturnsExistingID verifies the happy path:
// a duplicate hash within the 15-minute window is found and the existing ID
// and created_at are returned.
func TestDedupeCheck_HitWithinWindow_ReturnsExistingID(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory() // no TopicKey → non-topic-keyed
	saved := seedMemoryAt(t, st, brainID, m, t0)

	hash := NormalizedHash(m.Title, m.Content)
	windowStart := t0.Add(-DedupeWindow + 1*time.Minute) // 14 min ago: within window

	// Advance clock so "now" is 5 minutes after the save.
	st.SetClock(func() time.Time { return t0.Add(5 * time.Minute) })

	found, existingID, existingCreatedAt, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("DedupeCheck error: %v", err)
	}
	if !found {
		t.Fatal("expected found=true for hash within window")
	}
	if existingID != saved.ID {
		t.Errorf("existingID = %d, want %d", existingID, saved.ID)
	}
	if !existingCreatedAt.Equal(saved.CreatedAt) {
		t.Errorf("existingCreatedAt = %v, want %v", existingCreatedAt, saved.CreatedAt)
	}
}

// TestDedupeCheck_MissOutsideWindow verifies that a row whose created_at falls
// before the windowStart is not matched: same hash, same brain, but saved
// 16 minutes ago (outside the 15-minute window).
func TestDedupeCheck_MissOutsideWindow(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	seedMemoryAt(t, st, brainID, m, t0)

	hash := NormalizedHash(m.Title, m.Content)
	// windowStart is 1 minute after the save → row is outside the window.
	windowStart := t0.Add(1 * time.Minute)

	found, existingID, _, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("DedupeCheck error: %v", err)
	}
	if found {
		t.Errorf("expected found=false for row outside window; existingID=%d", existingID)
	}
}

// TestDedupeCheck_TopicKeySkipsDedupe verifies that a topic-keyed row with the
// same hash is invisible to DedupeCheck (topic_key IS NULL guard).
func TestDedupeCheck_TopicKeySkipsDedupe(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	m.TopicKey = "scene/playcanvas/inn-cellar" // topic-keyed row
	seedMemoryAt(t, st, brainID, m, t0)

	// Same hash as the topic-keyed row.
	hash := NormalizedHash(m.Title, m.Content)
	windowStart := t0.Add(-DedupeWindow)

	found, _, _, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("DedupeCheck error: %v", err)
	}
	if found {
		t.Error("topic-keyed row must not satisfy DedupeCheck")
	}
}

// TestDedupeCheck_SoftDeletedDoesNotDedupe verifies that a soft-deleted row
// with the matching hash is invisible (deleted_at IS NULL guard).
func TestDedupeCheck_SoftDeletedDoesNotDedupe(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	saved := seedMemoryAt(t, st, brainID, m, t0)

	// Soft-delete the row.
	if err := st.SoftDelete(ctx, brainID, saved.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	hash := NormalizedHash(m.Title, m.Content)
	windowStart := t0.Add(-DedupeWindow)

	found, _, _, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("DedupeCheck error: %v", err)
	}
	if found {
		t.Error("soft-deleted row must not satisfy DedupeCheck")
	}
}

// TestDedupeCheck_NullTopicKeyDedup verifies that a non-topic-keyed row
// (topic_key IS NULL, which is how insertNew stores it) IS matched by
// DedupeCheck. This is the positive contract for the NULL-topic-key path.
func TestDedupeCheck_NullTopicKeyDedup(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory() // TopicKey == "" → NULL stored by insertNew
	saved := seedMemoryAt(t, st, brainID, m, t0)

	hash := NormalizedHash(m.Title, m.Content)
	windowStart := t0.Add(-DedupeWindow)

	found, existingID, _, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("DedupeCheck error: %v", err)
	}
	if !found {
		t.Fatal("non-topic-keyed row must be visible to DedupeCheck")
	}
	if existingID != saved.ID {
		t.Errorf("existingID = %d, want %d", existingID, saved.ID)
	}
}

// TestDedupeCheck_DifferentBrainID_Ignored verifies brain-level isolation: a
// hash seeded under brain-a must NOT be found when DedupeCheck is called with
// brain-b's ID, even when the hash and windowStart are identical.
//
// This test would regress immediately if the brain_id = ? predicate were
// accidentally dropped from the DedupeCheck WHERE clause.
func TestDedupeCheck_DifferentBrainID_Ignored(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	// Create two separate brains.
	brainA, err := st.ResolveOrCreateBrainID(ctx, "brain-a")
	if err != nil {
		t.Fatalf("create brain-a: %v", err)
	}
	brainB, err := st.ResolveOrCreateBrainID(ctx, "brain-b")
	if err != nil {
		t.Fatalf("create brain-b: %v", err)
	}

	// Seed a non-topic-keyed observation under brain-a at t0.
	m := memory.Memory{
		Project: "brain-a",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeScenePattern,
		Title:   "Shared title",
		Content: "Shared content",
	}
	seedMemoryAt(t, st, brainA, m, t0)

	// Compute the same hash that brain-a's row carries.
	hash := NormalizedHash(m.Title, m.Content)
	// windowStart fully encompasses t0 — brain-a would return found=true.
	windowStart := t0.Add(-DedupeWindow + 1*time.Minute)

	// DedupeCheck under brain-b must NOT find brain-a's row.
	found, existingID, _, err := st.DedupeCheck(ctx, brainB, hash, windowStart)
	if err != nil {
		t.Fatalf("DedupeCheck error: %v", err)
	}
	if found {
		t.Errorf("brain isolation violated: DedupeCheck for brain-b returned found=true with existingID=%d (brain-a's row)", existingID)
	}
	if existingID != 0 {
		t.Errorf("expected existingID=0 for brain-b miss, got %d", existingID)
	}
}

// TestDedupeCheck_IdempotentReadsReturnSameRow verifies that two back-to-back
// DedupeCheck calls against the same row return identical results. This
// encodes the determinism guarantee that matters for the single-client stdio
// transport: because MCP messages arrive sequentially on one goroutine, the
// gap between DedupeCheck and Save will never be interleaved with another
// writer. The test does NOT spawn goroutines — true concurrent access is not
// a scenario Thoughtline needs to support (and would require the transactional
// Path B from design D11 instead).
func TestDedupeCheck_IdempotentReadsReturnSameRow(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := sampleMemory()
	saved := seedMemoryAt(t, st, brainID, m, t0)

	hash := NormalizedHash(m.Title, m.Content)
	windowStart := t0.Add(-DedupeWindow)

	found1, id1, ts1, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("first DedupeCheck: %v", err)
	}
	found2, id2, ts2, err := st.DedupeCheck(ctx, brainID, hash, windowStart)
	if err != nil {
		t.Fatalf("second DedupeCheck: %v", err)
	}

	if found1 != found2 {
		t.Errorf("consistency: found changed between calls (%v → %v)", found1, found2)
	}
	if id1 != id2 {
		t.Errorf("consistency: existingID changed between calls (%d → %d)", id1, id2)
	}
	if !ts1.Equal(ts2) {
		t.Errorf("consistency: existingCreatedAt changed between calls (%v → %v)", ts1, ts2)
	}
	// Confirm the row is actually found (sanity).
	if !found1 || id1 != saved.ID {
		t.Errorf("expected found=true, id=%d; got found=%v, id=%d", saved.ID, found1, id1)
	}
}
