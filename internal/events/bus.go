package events

import (
	"log/slog"
	"sync"
	"sync/atomic"
)

const defaultBufSize = 64

// subscription holds the channel and a closed guard for a single subscriber.
type subscription struct {
	ch     chan Event
	closed atomic.Bool
}

// Bus is a brain-scoped, non-blocking, in-process event bus.
// It is safe for concurrent use by multiple goroutines.
//
// Subscribers register by brain_id and receive only events whose BrainID
// matches their registration. Subscriptions with brain_id=0 receive all
// events (wildcard / "all brains" — useful for metrics and tests).
//
// Publish is always non-blocking: if a subscriber's buffer is full, the event
// is dropped for that subscriber, the drops counter is incremented, and a
// warning is logged via slog.
type Bus struct {
	mu      sync.RWMutex
	subs    map[int64][]*subscription
	bufSize int
	drops   atomic.Uint64
	log     *slog.Logger
}

// New creates a Bus with the default buffer size (64) per subscription.
func New(log *slog.Logger) *Bus {
	return NewWithBuffer(defaultBufSize)
}

// NewWithBuffer creates a Bus with the given buffer size per subscription.
// Use a small size in tests to force drops quickly; use a larger size in
// production to absorb event bursts.
func NewWithBuffer(bufSize int) *Bus {
	if bufSize <= 0 {
		bufSize = defaultBufSize
	}
	return &Bus{
		subs:    make(map[int64][]*subscription),
		bufSize: bufSize,
		log:     slog.Default(),
	}
}

// Subscribe registers a subscriber for the given brainID and returns a
// receive-only channel plus a cancel function.
//
// The cancel function is idempotent: calling it multiple times is safe.
// After cancel, the channel is closed and no further events are delivered.
// BrainID=0 subscribes to ALL brain events (wildcard).
func (b *Bus) Subscribe(brainID int64) (<-chan Event, func()) {
	sub := &subscription{
		ch: make(chan Event, b.bufSize),
	}

	b.mu.Lock()
	b.subs[brainID] = append(b.subs[brainID], sub)
	b.mu.Unlock()

	cancel := func() {
		// Idempotency guard: only one caller wins the CompareAndSwap.
		if !sub.closed.CompareAndSwap(false, true) {
			return // already cancelled
		}
		close(sub.ch)

		// Remove this subscription from the map.
		b.mu.Lock()
		defer b.mu.Unlock()
		list := b.subs[brainID]
		for i, s := range list {
			if s == sub {
				// Replace with last element and truncate (order doesn't matter).
				last := len(list) - 1
				list[i] = list[last]
				list[last] = nil
				b.subs[brainID] = list[:last]
				break
			}
		}
	}

	return sub.ch, cancel
}

// Publish delivers e to all subscribers of e.BrainID() and all wildcard
// subscribers (brainID=0). Each delivery is non-blocking: a full buffer drops
// the event for that subscriber only and increments the drops counter.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	// Collect subscriber lists: brain-specific + wildcard (0).
	specific := b.subs[e.BrainID()]
	wildcard := b.subs[0]
	// Take a shallow copy while under the read-lock to avoid holding it during
	// channel sends (which could block on the GC / scheduler).
	targets := make([]*subscription, 0, len(specific)+len(wildcard))
	targets = append(targets, specific...)
	if e.BrainID() != 0 {
		// Don't double-deliver if brainID IS 0 (already included above as specific).
		targets = append(targets, wildcard...)
	}
	b.mu.RUnlock()

	for _, sub := range targets {
		b.deliver(sub, e)
	}
}

// deliver attempts a non-blocking send. On buffer full it increments drops and
// logs a warning. It skips already-closed subscriptions.
func (b *Bus) deliver(sub *subscription, e Event) {
	if sub.closed.Load() {
		return
	}
	select {
	case sub.ch <- e:
		// delivered
	default:
		b.drops.Add(1)
		if b.log != nil {
			b.log.Warn("event bus drop",
				"kind", e.Kind(),
				"brain_id", e.BrainID(),
				"drops_total", b.drops.Load(),
			)
		}
	}
}

// Drops returns the total number of events dropped due to full subscriber
// buffers. Useful for telemetry and tests.
func (b *Bus) Drops() uint64 {
	return b.drops.Load()
}
