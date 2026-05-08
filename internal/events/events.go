// Package events provides an in-process pub/sub bus for Thoughtline.
//
// The Event interface is sealed: only the concrete types defined in this
// package implement it. internal/storage imports this package to construct and
// publish events; this package does NOT import internal/storage (no cycle).
package events

import "time"

// Event is the sealed base interface for all bus events.
// The unexported eventMarker() method prevents external types from implementing
// this interface, ensuring only the concrete types in this package are publishable.
type Event interface {
	BrainID()   int64
	Timestamp() time.Time
	Kind()      string
	eventMarker()
}

// MemoryCreated is emitted after a new memory is committed to the database.
type MemoryCreated struct {
	Brain    int64
	MemoryID int64
	SyncID   string
	At       time.Time
}

func (e MemoryCreated) BrainID() int64      { return e.Brain }
func (e MemoryCreated) Timestamp() time.Time { return e.At }
func (e MemoryCreated) Kind() string         { return "memory.created" }
func (e MemoryCreated) eventMarker()         {}

// MemoryUpdated is emitted after an existing memory is updated.
type MemoryUpdated struct {
	Brain    int64
	MemoryID int64
	SyncID   string
	At       time.Time
}

func (e MemoryUpdated) BrainID() int64      { return e.Brain }
func (e MemoryUpdated) Timestamp() time.Time { return e.At }
func (e MemoryUpdated) Kind() string         { return "memory.updated" }
func (e MemoryUpdated) eventMarker()         {}

// MemoryDeleted is emitted after a memory is soft-deleted.
type MemoryDeleted struct {
	Brain    int64
	MemoryID int64
	SyncID   string
	At       time.Time
}

func (e MemoryDeleted) BrainID() int64      { return e.Brain }
func (e MemoryDeleted) Timestamp() time.Time { return e.At }
func (e MemoryDeleted) Kind() string         { return "memory.deleted" }
func (e MemoryDeleted) eventMarker()         {}

// LinkCreated is emitted after a memory link is persisted.
type LinkCreated struct {
	Brain    int64
	LinkID   int64
	FromID   int64
	ToID     int64
	Relation string
	At       time.Time
}

func (e LinkCreated) BrainID() int64      { return e.Brain }
func (e LinkCreated) Timestamp() time.Time { return e.At }
func (e LinkCreated) Kind() string         { return "link.created" }
func (e LinkCreated) eventMarker()         {}

// LinkDeleted is emitted after a memory link is removed.
type LinkDeleted struct {
	Brain  int64
	LinkID int64
	At     time.Time
}

func (e LinkDeleted) BrainID() int64      { return e.Brain }
func (e LinkDeleted) Timestamp() time.Time { return e.At }
func (e LinkDeleted) Kind() string         { return "link.deleted" }
func (e LinkDeleted) eventMarker()         {}

// BrainConfigChanged is emitted when a brain's config_json is updated.
type BrainConfigChanged struct {
	Brain int64
	At    time.Time
}

func (e BrainConfigChanged) BrainID() int64      { return e.Brain }
func (e BrainConfigChanged) Timestamp() time.Time { return e.At }
func (e BrainConfigChanged) Kind() string         { return "brain.config_changed" }
func (e BrainConfigChanged) eventMarker()         {}
