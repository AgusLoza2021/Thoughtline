// Package links provides typed directed edges between memories within a brain.
// All operations are brain-scoped — cross-brain links are impossible by
// construction (rejected at the storage layer inside a transaction).
package links

import (
	"errors"
	"time"
)

// Relation is the closed set of edge types for memory_links.
type Relation string

const (
	RelSupersedes  Relation = "supersedes"
	RelContradicts Relation = "contradicts"
	RelRefines     Relation = "refines"
	RelDependsOn   Relation = "depends_on"
	RelReferences  Relation = "references"
	RelRelated     Relation = "related"
	RelDerivedFrom Relation = "derived_from"
)

// Valid reports whether r is one of the seven accepted relation values.
func (r Relation) Valid() bool {
	switch r {
	case RelSupersedes, RelContradicts, RelRefines,
		RelDependsOn, RelReferences, RelRelated, RelDerivedFrom:
		return true
	}
	return false
}

// Source tracks how a link was created.
type Source string

const (
	SourceManual   Source = "manual"
	SourceAuto     Source = "auto"
	SourceImported Source = "imported"
)

// Valid reports whether s is one of the three accepted source values.
func (s Source) Valid() bool {
	switch s {
	case SourceManual, SourceAuto, SourceImported:
		return true
	}
	return false
}

// Link is the in-memory representation of a memory_links row.
type Link struct {
	ID        int64
	BrainID   int64
	FromID    int64
	ToID      int64
	Relation  Relation
	Weight    float64
	Source    Source
	Note      string
	CreatedAt time.Time
}

// Sentinel errors returned by Store methods. Callers use errors.Is.
var (
	// ErrInvalidRelation is returned when an unknown relation string is passed.
	ErrInvalidRelation = errors.New("links: invalid relation")

	// ErrInvalidSource is returned when an unknown source string is passed.
	ErrInvalidSource = errors.New("links: invalid source")

	// ErrCrossBrainLink is returned when from_id and to_id belong to different brains.
	ErrCrossBrainLink = errors.New("links: cross-brain link rejected")

	// ErrSelfLoop is returned when from_id == to_id.
	ErrSelfLoop = errors.New("links: self-loop rejected")

	// ErrLinkExists is returned on a duplicate (from_id, to_id, relation) triple.
	ErrLinkExists = errors.New("links: link already exists")

	// ErrLinkNotFound is returned when a link id is not found or belongs to a
	// different brain.
	ErrLinkNotFound = errors.New("links: link not found")

	// ErrEndpointNotFound is returned when from_id or to_id does not refer to
	// an active (non-deleted) memory.
	ErrEndpointNotFound = errors.New("links: endpoint memory not found")
)
