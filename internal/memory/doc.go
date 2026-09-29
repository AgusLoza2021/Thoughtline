// Package memory defines the core domain types for Thoughtline.
//
// Status: archived with the v0.1.0 MCP server. This package is the domain layer
// of the retired engine. It still compiles and its tests still pass, and it is
// kept as the record of what that server enforced. It is not the product, and it
// is not the normative vocabulary.
//
// The live catalogue is docs/design/memory-domain.md, and it is maintained
// independently of this package. The two have deliberately diverged: the
// catalogue carries fourteen types in two tiers, models a `global` scope, and
// permits a dot in `topic_key` for version numbers. This copy knows none of that,
// on purpose — it is frozen at what the v0.1.0 server accepted.
//
// A Memory is a single observation persisted by the user (via an AI assistant
// calling tl_save). Each Memory carries:
//
//   - Type        — one of the eleven values this package closes the set to; see
//                   AllTypes.
//   - Scope       — project (default; tied to a project id) or personal
//                   (cross-project, per-developer ergonomics). Only the two
//                   values the v0.1.0 server modelled; the live rules also allow
//                   `global`, which this package does not represent.
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
// It is kept for the record: the split, the sentinel errors and the frozen type
// set are what the retired server shipped.
package memory
