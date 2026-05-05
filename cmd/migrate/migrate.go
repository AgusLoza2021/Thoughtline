// Package migrate implements the one-shot migration from Engram's SQLite
// database to Thoughtline's storage layer. It is intentionally a standalone
// binary (cmd/migrate/main.go) so it can be built, audited, and discarded
// without touching the MCP server's public API surface.
//
// Public types in this file are the shared vocabulary used by reader.go,
// mapper.go, writer.go, and main.go. No logic lives here — types only.
package migrate

// EngramObservation is the raw row read from the Engram observations table.
// Columns that Engram stores but Thoughtline has no equivalent for
// (tool_name, duplicate_count, last_seen_at) are read and discarded by the
// reader — they never appear here.
type EngramObservation struct {
	SyncID         string
	Type           string  // open set: bugfix|decision|architecture|pattern|config|preference|discovery|manual
	Title          string
	Content        string
	Project        string
	Scope          string
	TopicKey       string
	Tags           []string // Engram has no tags column; always nil. Reserved for future.
	NormalizedHash string   // stored but not forwarded — storage.Save() recomputes it
	RevisionCount  int
	CreatedAt      string  // ISO 8601
	UpdatedAt      string  // ISO 8601
	DeletedAt      *string // nil = active row
}

// RowResult captures what happened to a single Engram row during migration.
// The full slice of RowResults is written to the structured log file; stdout
// shows only aggregate counters.
type RowResult struct {
	SyncID string
	// Action is one of: "created", "skipped-deleted", "skipped-duplicate",
	// "skipped-topic-collision", "error".
	Action string
	// Reason is populated when Action is "error" or any "skipped-*" variant.
	Reason string
}

// Summary is the aggregate result of a migration run. Run() always returns a
// Summary even when individual rows errored — per-row errors are non-fatal.
type Summary struct {
	Total              int
	Created            int
	SkippedDeleted     int // soft-deleted source rows; not migrated by spec
	SkippedDuplicate   int // already in destination by sync_id
	SkippedTopicCol    int // (project, topic_key) collision with existing Thoughtline row
	Errors             int // rows that failed mapping or storage; do not abort the run
	Truncations        int // rows whose title was truncated at 200 runes
	Rows               []RowResult
}

// Config holds the CLI flags resolved by main.go before calling Run.
type Config struct {
	// Source is the absolute path to engram.db (read-only).
	Source string
	// Dest is the absolute path to thoughtline.db (read-write).
	Dest string
	// DryRun, when true, performs all reads and mapping but skips all writes.
	DryRun bool
	// Verbose, when true, causes each row result to be emitted to the logger
	// as it is processed (not just at the end).
	Verbose bool
}

// Logger is the interface mapper and writer functions use so they stay
// pure and testable. StructuredLogger and NopLogger both satisfy it.
// Log fields are key=value pairs; callers pass them as a flat map.
type Logger interface {
	Log(level, msg string, fields map[string]string)
}
