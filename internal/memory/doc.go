// Package memory defines the core domain types for Thoughtline.
//
// Status: live. This package is the domain layer, and the server validates every
// save through it: a save carrying a type outside the eleven values below is
// rejected rather than stored.
//
// The catalogue in docs/design/memory-domain.md is broader than this package, and
// the two currently disagree. The catalogue carries fourteen types in two tiers,
// documents a `global` scope, and permits a dot in `topic_key` for version
// numbers; none of the three is accepted here.
//
// A Memory is a single observation persisted by the user (via an AI assistant
// calling tl_save). Each Memory carries:
//
//   - Type        — one of the eleven values this package closes the set to; see
//                   AllTypes.
//   - Scope       — project (default; tied to a project id) or personal
//                   (cross-project, per-developer ergonomics). Only those two
//                   are accepted; the catalogue also documents `global`, which
//                   this package does not represent.
//   - TopicKey    — optional stable key for evolving topics. When set, tl_save
//                   upserts: same project + same topic_key replaces the prior
//                   row, preserving created_at and bumping revision_count.
//   - Project     — string identifier of the active project (typically the
//                   working directory's basename, but explicitly configurable).
//   - Title       — short, searchable headline. Imperative form preferred
//                   ("Fix N+1 in InventoryList").
//   - Content     — the body. Markdown. No length cap is enforced at the
//                   domain layer; storage layer applies a soft limit and
//                   returns a clear error rather than truncating silently.
//   - CreatedAt   — first time this memory (or its topic) appeared.
//   - UpdatedAt   — last time this memory (or its topic) was upserted.
//   - Revision    — number of times the topic has been re-saved (0 for first
//                   save, incremented on every upsert).
//
// This package contains pure data types and validation. No I/O, no persistence
// concerns — those belong in internal/storage. The split keeps the domain
// reusable across storage backends.
//
package memory
