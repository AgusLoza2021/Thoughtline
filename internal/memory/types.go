package memory

import (
	"errors"
	"time"
)

// Type is the kind of memory being stored. The set is closed by design — see
// docs/design/memory-domain.md for the rationale and per-type usage.
type Type string

const (
	TypeGameDesignDecision Type = "game-design-decision"
	TypeScenePattern       Type = "scene-pattern"
	TypeAssetReference     Type = "asset-reference"
	TypePerfGotcha         Type = "perf-gotcha"
	TypePipelineStep       Type = "pipeline-step"
	TypeScriptPattern      Type = "script-pattern"
	TypeBugfix             Type = "bugfix"
	TypeConvention         Type = "convention"
	TypePreference         Type = "preference"
)

// AllTypes returns the canonical type set in stable order.
func AllTypes() []Type {
	return []Type{
		TypeGameDesignDecision,
		TypeScenePattern,
		TypeAssetReference,
		TypePerfGotcha,
		TypePipelineStep,
		TypeScriptPattern,
		TypeBugfix,
		TypeConvention,
		TypePreference,
	}
}

// Valid reports whether t is one of the catalogued types.
func (t Type) Valid() bool {
	for _, allowed := range AllTypes() {
		if t == allowed {
			return true
		}
	}
	return false
}

// Scope governs whether a memory belongs to a specific project or travels
// with the developer across projects.
type Scope string

const (
	ScopeProject  Scope = "project"
	ScopePersonal Scope = "personal"
)

// Valid reports whether s is one of the supported scopes.
func (s Scope) Valid() bool {
	return s == ScopeProject || s == ScopePersonal
}

// Memory is the canonical in-memory representation of a stored observation.
// Storage maps this to/from the SQL row; the server (MCP) maps this to/from
// the JSON request payload.
type Memory struct {
	// ID is the local autoincrement primary key. Zero for unsaved memories.
	ID int64
	// SyncID is a UUIDv7 stable across upserts. Set by storage on first save.
	SyncID string
	// Project is the project identifier the memory belongs to. Required.
	Project string
	// Scope governs cross-project visibility.
	Scope Scope
	// Type categorises the memory.
	Type Type
	// TopicKey, when non-empty, makes (Project, TopicKey) the upsert key.
	TopicKey string
	// Title is a short, searchable headline.
	Title string
	// Content is the markdown body.
	Content string
	// Tags are free-form labels (engine name, platform, etc.). Optional.
	Tags []string
	// NormalizedHash is a content fingerprint used for noop detection. Set by
	// storage on save; callers should leave this zero.
	NormalizedHash string
	// RevisionCount is the number of upserts on this topic. 0 for first save.
	RevisionCount int
	// CreatedAt is the first time this memory (or its topic) was saved.
	CreatedAt time.Time
	// UpdatedAt is the last time this memory (or its topic) was saved.
	UpdatedAt time.Time
	// DeletedAt is non-nil for soft-deleted memories.
	DeletedAt *time.Time
	// SessionID optionally associates this memory with a Session (UUIDv7).
	// Empty when the memory was saved outside any session. Set via tl_save's
	// optional session_id argument.
	SessionID string
}

// Errors returned by Validate. Tests assert against these directly.
var (
	ErrInvalidType                 = errors.New("memory: invalid type")
	ErrInvalidScope                = errors.New("memory: invalid scope")
	ErrPreferenceMustBePersonal    = errors.New("memory: preference type requires personal scope")
	ErrNonPreferenceMustBeProject  = errors.New("memory: non-preference types require project scope")
	ErrEmptyProject                = errors.New("memory: project must not be empty")
	ErrEmptyTitle                  = errors.New("memory: title must not be empty")
	ErrTitleTooLong                = errors.New("memory: title exceeds 200 characters")
	ErrEmptyContent                = errors.New("memory: content must not be empty")
	ErrContentTooLong              = errors.New("memory: content exceeds maximum bytes")
	ErrInvalidTopicKey             = errors.New("memory: invalid topic_key format")
	ErrInvalidTag                  = errors.New("memory: invalid tag format")
)

// Limits — exported so callers can show meaningful validation messages.
const (
	MaxTitleChars   = 200
	MaxContentBytes = 64 * 1024
	MaxTopicKeyLen  = 128
	MaxTagLen       = 40
)
