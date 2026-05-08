package storage

import (
	"context"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/events"
	"github.com/AgusLoza2021/Thoughtline/internal/memory"
)

// busTestSetup creates storage + bus + a brainID and returns them along with
// a cleanup cancel for the subscriber channel.
func busTestSetup(t *testing.T) (*Storage, *events.Bus, int64, <-chan events.Event, func()) {
	t.Helper()
	st := newTestStorage(t)
	bus := events.NewWithBuffer(16)
	st.SetBus(bus)
	brainID := defaultBrainID(t, st)
	ch, cancel := bus.Subscribe(brainID)
	return st, bus, brainID, ch, cancel
}

// receiveWithTimeout reads one event from ch or fails after 1 s.
func receiveWithTimeout(t *testing.T, ch <-chan events.Event) events.Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for bus event")
		return nil
	}
}

// expectNoEvent asserts that no event is delivered within a short window.
func expectNoEvent(t *testing.T, ch <-chan events.Event) {
	t.Helper()
	select {
	case e := <-ch:
		t.Fatalf("expected no event but got %T (brain_id=%d)", e, e.BrainID())
	case <-time.After(50 * time.Millisecond):
		// good — nothing received
	}
}

// TestStorage_EmitsMemoryCreated — task 6.11
// SetBus → Save (new) → MemoryCreated received with correct memory_id.
func TestStorage_EmitsMemoryCreated(t *testing.T) {
	st, _, brainID, ch, cancel := busTestSetup(t)
	defer cancel()
	ctx := context.Background()

	saved, action, err := st.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if action != ActionCreated {
		t.Fatalf("want ActionCreated, got %s", action)
	}

	e := receiveWithTimeout(t, ch)
	mc, ok := e.(events.MemoryCreated)
	if !ok {
		t.Fatalf("want events.MemoryCreated, got %T", e)
	}
	if mc.MemoryID != saved.ID {
		t.Fatalf("memory_id: want %d, got %d", saved.ID, mc.MemoryID)
	}
	if mc.BrainID() != brainID {
		t.Fatalf("brain_id: want %d, got %d", brainID, mc.BrainID())
	}
	if mc.Timestamp().IsZero() {
		t.Fatal("timestamp must be non-zero")
	}
}

// TestStorage_EmitsMemoryUpdated — Save with topic_key twice (different content)
// → second Save emits MemoryUpdated.
func TestStorage_EmitsMemoryUpdated(t *testing.T) {
	st, _, brainID, ch, cancel := busTestSetup(t)
	defer cancel()
	ctx := context.Background()

	m := sampleMemory()
	m.TopicKey = "test-topic"

	// First save → MemoryCreated
	_, _, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	receiveWithTimeout(t, ch) // consume the MemoryCreated

	// Second save with different content → MemoryUpdated
	m.Content = "updated content — different hash"
	saved, action, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("second save: %v", err)
	}
	if action != ActionUpdated {
		t.Fatalf("want ActionUpdated, got %s", action)
	}

	e := receiveWithTimeout(t, ch)
	mu, ok := e.(events.MemoryUpdated)
	if !ok {
		t.Fatalf("want events.MemoryUpdated, got %T", e)
	}
	if mu.MemoryID != saved.ID {
		t.Fatalf("memory_id: want %d, got %d", saved.ID, mu.MemoryID)
	}
}

// TestStorage_NoEventOnNoop — Save with identical content (NoOp) emits nothing.
func TestStorage_NoEventOnNoop(t *testing.T) {
	st, _, brainID, ch, cancel := busTestSetup(t)
	defer cancel()
	ctx := context.Background()

	m := sampleMemory()
	m.TopicKey = "noop-topic"

	// First save
	_, _, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("first save: %v", err)
	}
	receiveWithTimeout(t, ch) // consume MemoryCreated

	// Re-save with identical content → NoOp
	_, action, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("noop save: %v", err)
	}
	if action != ActionNoop {
		t.Fatalf("want ActionNoop, got %s", action)
	}
	expectNoEvent(t, ch)
}

// TestStorage_EmitsMemoryDeleted — SoftDelete → MemoryDeleted received.
func TestStorage_EmitsMemoryDeleted(t *testing.T) {
	st, _, brainID, ch, cancel := busTestSetup(t)
	defer cancel()
	ctx := context.Background()

	saved, _, err := st.Save(ctx, brainID, sampleMemory())
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	receiveWithTimeout(t, ch) // consume MemoryCreated

	if err := st.SoftDelete(ctx, brainID, saved.ID); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	e := receiveWithTimeout(t, ch)
	md, ok := e.(events.MemoryDeleted)
	if !ok {
		t.Fatalf("want events.MemoryDeleted, got %T", e)
	}
	if md.MemoryID != saved.ID {
		t.Fatalf("memory_id: want %d, got %d", saved.ID, md.MemoryID)
	}
	if md.BrainID() != brainID {
		t.Fatalf("brain_id: want %d, got %d", brainID, md.BrainID())
	}
}

// TestStorage_NilBus_NoPanic — nil bus (default) means all operations succeed
// without panic even when no SetBus was called.
func TestStorage_NilBus_NoPanic(t *testing.T) {
	st := newTestStorage(t) // no SetBus — bus is nil
	ctx := context.Background()
	brainID := defaultBrainID(t, st)

	m := memory.Memory{
		Project: "enchanted-inn",
		Scope:   memory.ScopeProject,
		Type:    memory.TypeScenePattern,
		Title:   "nil bus test",
		Content: "must not panic",
	}

	saved, _, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("save with nil bus: %v", err)
	}
	if err := st.SoftDelete(ctx, brainID, saved.ID); err != nil {
		t.Fatalf("delete with nil bus: %v", err)
	}
}

// TestStorage_EmitsMemoryCreated_TopicKey — upsert path (topic_key) also
// emits MemoryCreated on first insert.
func TestStorage_EmitsMemoryCreated_TopicKey(t *testing.T) {
	st, _, brainID, ch, cancel := busTestSetup(t)
	defer cancel()
	ctx := context.Background()

	m := sampleMemory()
	m.TopicKey = "upsert-topic"

	saved, action, err := st.Save(ctx, brainID, m)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if action != ActionCreated {
		t.Fatalf("want ActionCreated, got %s", action)
	}

	e := receiveWithTimeout(t, ch)
	mc, ok := e.(events.MemoryCreated)
	if !ok {
		t.Fatalf("want events.MemoryCreated, got %T", e)
	}
	if mc.MemoryID != saved.ID {
		t.Fatalf("memory_id: want %d, got %d", saved.ID, mc.MemoryID)
	}
}

// TestStorage_NoEventOnFailedSave — task 6.12
// When Save returns an error, no event must be emitted.
// We use brainID=0 (ErrBrainRequired) and brainID mismatch (ErrBrainMismatch)
// as two reliable error triggers that return before any DB write.
func TestStorage_NoEventOnFailedSave(t *testing.T) {
	st, bus, brainID, _, cancel := busTestSetup(t)
	defer cancel()
	ctx := context.Background()

	// Subscribe to the wildcard (brainID=0) channel to catch any stray events.
	allCh, cancelAll := bus.Subscribe(0)
	defer cancelAll()

	m := sampleMemory()

	// Case 1: brainID=0 → ErrBrainRequired; no DB write, no event.
	_, _, err := st.Save(ctx, 0, m)
	if err == nil {
		t.Fatal("expected ErrBrainRequired for brainID=0")
	}
	expectNoEvent(t, allCh)

	// Case 2: m.BrainID set to a different brain → ErrBrainMismatch; no event.
	mismatch := m
	mismatch.BrainID = brainID + 999
	_, _, err = st.Save(ctx, brainID, mismatch)
	if err == nil {
		t.Fatal("expected ErrBrainMismatch")
	}
	expectNoEvent(t, allCh)
}
