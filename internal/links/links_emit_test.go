package links_test

import (
	"context"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/events"
	"github.com/AgusLoza2021/Thoughtline/internal/links"
)

// TestLinks_BusEmitOnCreate verifies that a LinkCreated event is published to
// the bus after a successful link creation.
func TestLinks_BusEmitOnCreate(t *testing.T) {
	s := openDB(t)
	db := s.DB()
	brainA := createBrain(t, db, "emit-brain-a")
	m1 := createMemory(t, s, brainA.ID, "emit mem 1")
	m2 := createMemory(t, s, brainA.ID, "emit mem 2")

	bus := events.New(nil)
	ch, unsub := bus.Subscribe(brainA.ID)
	defer unsub()

	ls := links.New(db)
	ls.SetBus(bus)

	ctx := context.Background()
	lk, err := ls.Create(ctx, brainA.ID, m1.ID, m2.ID, links.RelRefines, links.SourceManual, 1.0, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	select {
	case ev := <-ch:
		lc, ok := ev.(events.LinkCreated)
		if !ok {
			t.Fatalf("expected LinkCreated, got %T", ev)
		}
		if lc.LinkID != lk.ID {
			t.Errorf("LinkCreated.LinkID = %d, want %d", lc.LinkID, lk.ID)
		}
		if lc.Brain != brainA.ID {
			t.Errorf("LinkCreated.Brain = %d, want %d", lc.Brain, brainA.ID)
		}
		if lc.FromID != m1.ID {
			t.Errorf("LinkCreated.FromID = %d, want %d", lc.FromID, m1.ID)
		}
		if lc.ToID != m2.ID {
			t.Errorf("LinkCreated.ToID = %d, want %d", lc.ToID, m2.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for LinkCreated event")
	}
}

// TestLinks_BusEmitOnDelete verifies that a LinkDeleted event is published to
// the bus after a successful link deletion.
func TestLinks_BusEmitOnDelete(t *testing.T) {
	s := openDB(t)
	db := s.DB()
	brainA := createBrain(t, db, "emit-del-brain")
	m1 := createMemory(t, s, brainA.ID, "del mem 1")
	m2 := createMemory(t, s, brainA.ID, "del mem 2")

	bus := events.New(nil)
	ch, unsub := bus.Subscribe(brainA.ID)
	defer unsub()

	ls := links.New(db)
	ls.SetBus(bus)

	ctx := context.Background()
	lk, err := ls.Create(ctx, brainA.ID, m1.ID, m2.ID, links.RelRelated, links.SourceManual, 1.0, "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	// Drain the LinkCreated event.
	select {
	case <-ch:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for LinkCreated drain")
	}

	if err := ls.Delete(ctx, brainA.ID, lk.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	select {
	case ev := <-ch:
		ld, ok := ev.(events.LinkDeleted)
		if !ok {
			t.Fatalf("expected LinkDeleted, got %T", ev)
		}
		if ld.LinkID != lk.ID {
			t.Errorf("LinkDeleted.LinkID = %d, want %d", ld.LinkID, lk.ID)
		}
		if ld.Brain != brainA.ID {
			t.Errorf("LinkDeleted.Brain = %d, want %d", ld.Brain, brainA.ID)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for LinkDeleted event")
	}
}

// TestLinks_BusNilSafe verifies that Create and Delete work correctly when
// no bus is set (nil bus is safe).
func TestLinks_BusNilSafe(t *testing.T) {
	s := openDB(t)
	db := s.DB()
	brainA := createBrain(t, db, "nil-bus-brain")
	m1 := createMemory(t, s, brainA.ID, "nil mem 1")
	m2 := createMemory(t, s, brainA.ID, "nil mem 2")

	ls := links.New(db) // no SetBus call

	ctx := context.Background()
	lk, err := ls.Create(ctx, brainA.ID, m1.ID, m2.ID, links.RelRelated, links.SourceManual, 1.0, "")
	if err != nil {
		t.Fatalf("Create without bus: %v", err)
	}
	if err := ls.Delete(ctx, brainA.ID, lk.ID); err != nil {
		t.Fatalf("Delete without bus: %v", err)
	}
}
