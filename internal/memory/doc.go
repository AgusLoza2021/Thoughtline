// Package memory defines the core domain types for Thoughtline.
//
// A Memory is a single observation persisted by the user (via an AI assistant
// calling tl_save). Each Memory carries:
//
//   - Type        — one of the values in the memory taxonomy (see
//                   docs/design/memory-domain.md). Examples: game-design-decision,
//                   scene-pattern, asset-reference, perf-gotcha, pipeline-step,
//                   bugfix, convention.
//   - Scope       — project (default; tied to a project id) or personal
//                   (cross-project, per-developer ergonomics).
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
// Status: skeleton. Concrete types arrive in milestone M1.
package memory
