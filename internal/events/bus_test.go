package events_test

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/events"
)

// helper: drain ch for up to dur, return all received events.
func drainFor(ch <-chan events.Event, dur time.Duration) []events.Event {
	var out []events.Event
	timer := time.NewTimer(dur)
	defer timer.Stop()
	for {
		select {
		case e, ok := <-ch:
			if !ok {
				return out
			}
			out = append(out, e)
		case <-timer.C:
			return out
		}
	}
}

// TestBus_Subscribe_ReceivesMatchingBrain — task 6.1
// Subscribe to brainA, publish MemoryCreated for brainA, assert received.
func TestBus_Subscribe_ReceivesMatchingBrain(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)

	const brainA int64 = 1
	ch, cancel := bus.Subscribe(brainA)
	defer cancel()

	now := time.Now()
	bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: 42, SyncID: "s1", At: now})

	select {
	case e := <-ch:
		mc, ok := e.(events.MemoryCreated)
		if !ok {
			t.Fatalf("want MemoryCreated, got %T", e)
		}
		if mc.BrainID() != brainA {
			t.Fatalf("brain_id: want %d, got %d", brainA, mc.BrainID())
		}
		if mc.MemoryID != 42 {
			t.Fatalf("memory_id: want 42, got %d", mc.MemoryID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

// TestBus_BrainIsolation — task 6.4
// S1→brainA, S2→brainB; publish to brainA; S1 receives, S2 gets nothing.
func TestBus_BrainIsolation(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)

	const (
		brainA int64 = 1
		brainB int64 = 2
	)
	chA, cancelA := bus.Subscribe(brainA)
	defer cancelA()
	chB, cancelB := bus.Subscribe(brainB)
	defer cancelB()

	bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: 10, At: time.Now()})

	// S1 must receive
	select {
	case e := <-chA:
		if e.BrainID() != brainA {
			t.Fatalf("wrong brain_id: %d", e.BrainID())
		}
	case <-time.After(time.Second):
		t.Fatal("S1 timed out")
	}

	// S2 must not receive within a short window
	got := drainFor(chB, 50*time.Millisecond)
	if len(got) != 0 {
		t.Fatalf("S2 received %d events for brainB, want 0", len(got))
	}
}

// TestBus_SlowSubscriberDrop — task 6.5
// Never-draining subscriber fills buffer; excess publishes increment drops.
// A fast second subscriber (larger buffer) receives all events without blocking.
func TestBus_SlowSubscriberDrop(t *testing.T) {
	t.Parallel()
	const bufSize = 64
	const publishCount = 100

	const brainA int64 = 1

	// Create bus with bufSize=64. Slow sub never reads so its buffer fills and
	// causes drops. Fast sub drains concurrently during publishing so its buffer
	// never fills — verifying that the slow sub's drops don't affect it.
	smallBus := events.NewWithBuffer(bufSize)

	// Slow: never reads
	_, cancelSlow := smallBus.Subscribe(brainA)
	defer cancelSlow()

	// Fast: drains concurrently via goroutine
	chFast, cancelFast := smallBus.Subscribe(brainA)
	defer cancelFast()

	var received atomic.Int64
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range chFast {
			received.Add(1)
		}
	}()

	for i := 0; i < publishCount; i++ {
		smallBus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: int64(i), At: time.Now()})
	}

	// Give the fast consumer goroutine a moment to drain remaining buffered events.
	time.Sleep(50 * time.Millisecond)

	// Cancel both subs so the drain goroutine sees channel close.
	cancelFast()
	cancelSlow()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drain goroutine timed out")
	}

	// Fast subscriber received all 100 events.
	if got := received.Load(); got != publishCount {
		t.Fatalf("fast subscriber: want %d events, got %d", publishCount, got)
	}

	// Slow subscriber should have drops >= publishCount - bufSize.
	drops := smallBus.Drops()
	if drops < uint64(publishCount-bufSize) {
		t.Fatalf("drops: want >= %d, got %d", publishCount-bufSize, drops)
	}
}

// TestBus_Unsubscribe — task 6.7
// After cancel, channel is closed; further publishes don't panic.
// Another subscriber's drops are unchanged.
func TestBus_Unsubscribe(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)
	const brainA int64 = 1

	ch, cancel := bus.Subscribe(brainA)

	// Publish 3 events
	for i := 0; i < 3; i++ {
		bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: int64(i), At: time.Now()})
	}

	// Read all 3
	for i := 0; i < 3; i++ {
		select {
		case <-ch:
		case <-time.After(time.Second):
			t.Fatalf("timed out reading event %d", i)
		}
	}

	dropsBefore := bus.Drops()

	// Cancel unsubscribes
	cancel()

	// Channel should be closed
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed after cancel")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel close")
	}

	// Publish after cancel: must not panic
	bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: 99, At: time.Now()})

	// Drops unchanged (no other subscriber to receive for)
	if bus.Drops() != dropsBefore {
		t.Fatalf("drops changed after unsubscribe and publish with no other subs")
	}
}

// TestBus_IdempotentCancel — calling cancel twice must not panic.
func TestBus_IdempotentCancel(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)
	_, cancel := bus.Subscribe(1)
	cancel()
	cancel() // must not panic or double-close
}

// TestBus_ConcurrentPublishSubscribe — task 6.9 (race test)
// 10 subscribers + 100 publishers; -race must pass.
func TestBus_ConcurrentPublishSubscribe(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(64)
	const brainA int64 = 1
	const numSubs = 10
	const numPubs = 100

	cancels := make([]func(), numSubs)
	for i := range cancels {
		_, cancels[i] = bus.Subscribe(brainA)
	}

	var wg sync.WaitGroup
	for i := 0; i < numPubs; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: int64(n), At: time.Now()})
		}(i)
	}
	wg.Wait()

	for _, c := range cancels {
		c()
	}
	// No race, no panic — test passes by reaching here cleanly.
}

// TestBus_EmptySubscriberList — publishing to brain with no subs is a no-op.
func TestBus_EmptySubscriberList(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)
	// Must not panic
	bus.Publish(events.MemoryCreated{Brain: 999, MemoryID: 1, At: time.Now()})
	if bus.Drops() != 0 {
		t.Fatalf("drops should be 0 when no subscribers, got %d", bus.Drops())
	}
}

// TestBus_MultipleSubscribersSameBrain — all receive each event.
func TestBus_MultipleSubscribersSameBrain(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)
	const brainA int64 = 1

	ch1, cancel1 := bus.Subscribe(brainA)
	defer cancel1()
	ch2, cancel2 := bus.Subscribe(brainA)
	defer cancel2()

	bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: 7, At: time.Now()})

	for i, ch := range []<-chan events.Event{ch1, ch2} {
		select {
		case e := <-ch:
			if e.(events.MemoryCreated).MemoryID != 7 {
				t.Fatalf("sub %d: wrong memory_id", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("sub %d timed out", i)
		}
	}
}

// TestBus_GoroutineLeak — subscribe+cancel N times; goroutine count stable.
func TestBus_GoroutineLeak(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(4)
	const brainA int64 = 1

	baseline := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		_, cancel := bus.Subscribe(brainA)
		cancel()
	}

	// Give scheduler a moment to clean up.
	time.Sleep(10 * time.Millisecond)
	runtime.GC()
	time.Sleep(10 * time.Millisecond)

	after := runtime.NumGoroutine()
	const tolerance = 5
	if after > baseline+tolerance {
		t.Fatalf("goroutine leak: before %d, after %d (tolerance %d)", baseline, after, tolerance)
	}
}

// TestBus_AllEventTypes — verify all event types satisfy the Event interface.
func TestBus_AllEventTypes(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(8)
	const brainA int64 = 1
	ch, cancel := bus.Subscribe(brainA)
	defer cancel()

	now := time.Now()

	toPublish := []events.Event{
		events.MemoryCreated{Brain: brainA, MemoryID: 1, At: now},
		events.MemoryUpdated{Brain: brainA, MemoryID: 1, At: now},
		events.MemoryDeleted{Brain: brainA, MemoryID: 1, At: now},
		events.LinkCreated{Brain: brainA, LinkID: 1, FromID: 1, ToID: 2, Relation: "related", At: now},
		events.LinkDeleted{Brain: brainA, LinkID: 1, At: now},
		events.BrainConfigChanged{Brain: brainA, At: now},
	}

	for _, e := range toPublish {
		bus.Publish(e)
	}

	got := drainFor(ch, time.Second)
	if len(got) != len(toPublish) {
		t.Fatalf("want %d events, got %d", len(toPublish), len(got))
	}
}

// TestBus_100Publishers_TotalReceived — aggregate totals match (modulo drops).
func TestBus_100Publishers_TotalReceived(t *testing.T) {
	t.Parallel()
	bus := events.NewWithBuffer(256)
	const brainA int64 = 1

	ch, cancel := bus.Subscribe(brainA)
	defer cancel()

	const numPubs = 100
	var wg sync.WaitGroup
	for i := 0; i < numPubs; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			bus.Publish(events.MemoryCreated{Brain: brainA, MemoryID: int64(n), At: time.Now()})
		}(i)
	}
	wg.Wait()

	got := drainFor(ch, time.Second)
	drops := bus.Drops()
	total := uint64(len(got)) + drops
	if total != numPubs {
		t.Fatalf("received %d + drops %d = %d, want %d", len(got), drops, total, numPubs)
	}
}

// Ensure atomic usage is correct with race detector.
var _ atomic.Uint64
