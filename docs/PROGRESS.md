# Progress

Single source of truth for "what's done, what's next, what's blocking publishing." Updated at the end of every working session.

---

## Current state — 2026-04-28

**Milestone M0 (Bootstrap): 🟢 done.**

Everything below is committed in the working tree and ready to push to a public GitHub repo, pending the [pre-publish TODOs](#pre-publish-todos) at the bottom of this file.

### What got done this session

- Cloned [Engram](https://github.com/Gentleman-Programming/engram) (depth=50) into `_engram-research/` *outside* the project tree, so it never accidentally gets committed.
- Project skeleton on Desktop: `cmd/thoughtline/`, `internal/{server,storage,memory}/` with package `doc.go` files, `go.mod` targeting Go 1.25, `LICENSE` (MIT, with attribution to Engram), `.gitignore`.
- **Architectural reconnaissance** (delegated to a sub-agent for context isolation) produced three deep-dive docs in [`research/`](research/):
  - [`engram-anatomy.md`](research/engram-anatomy.md) — high-level repo map.
  - [`flow-mem-save.md`](research/flow-mem-save.md) — `mem_save` traced end-to-end.
  - [`flow-mem-search.md`](research/flow-mem-search.md) — search internals; the load-bearing one.
- **Critical insight from research**: Engram has reserved schema columns for embeddings but never uses them. Search is 100% FTS5 + BM25 with a topic-key shortcut. This significantly simplified our M1/M2 plan.
- ADRs:
  - [`0001-architecture-baseline.md`](decisions/0001-architecture-baseline.md) — copy Engram's pattern set, customize taxonomy, build parallel (not fork).
  - [`0002-search-strategy-fts5-first.md`](decisions/0002-search-strategy-fts5-first.md) — FTS5 + BM25 in v1, embeddings deferred to M5 with reserved schema columns.
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — components, data flows (mermaid), schema sketch.
- [`design/memory-domain.md`](design/memory-domain.md) — full memory type taxonomy with examples and validation rules.
- README, CHANGELOG, CONTRIBUTING, example MCP config, basic CI workflow (build + vet + test on Linux/macOS/Windows).

### What's NOT done (intentionally)

- No tools registered yet. `main()` only prints a skeleton banner.
- No SQL schema executed. `internal/storage/doc.go` describes the shape; implementation lands in M1.
- No tests. The first tests arrive with the first real code in M1.

---

## Next session — Milestone M1 (Save)

**Goal**: `tl_save` works end-to-end.

### M1 definition of done

- [ ] Storage layer:
  - [ ] `internal/storage/schema.go` with the DDL from [`ARCHITECTURE.md`](ARCHITECTURE.md), executed at startup if missing.
  - [ ] Forward-only migration runner (single-version for now; framework for future versions).
  - [ ] `Insert(ctx, Memory) (int64, error)` — for memories without a topic key.
  - [ ] `UpsertByTopicKey(ctx, Memory) (int64, action, error)` where `action ∈ {created, updated}`.
  - [ ] FTS5 virtual table + triggers verified to stay in sync (table-driven test).
- [ ] Domain layer:
  - [ ] `internal/memory/types.go` with `Memory`, `Type`, `Scope`, `TopicKey` types.
  - [ ] `internal/memory/validate.go` with all rules from `design/memory-domain.md`.
  - [ ] Unit tests for every validation rule (positive + negative).
- [ ] Server layer:
  - [ ] `internal/server/server.go` boots an mcp-go stdio server.
  - [ ] `internal/server/tl_save.go` registers `tl_save` with full JSON Schema and a thin handler.
  - [ ] Integration test: spin up an in-process MCP transport, call `tl_save` twice with the same `topic_key`, assert second call reports `action: updated` and `revision_count: 1`.
- [ ] CLI:
  - [ ] `cmd/thoughtline/main.go` resolves `THOUGHTLINE_HOME` (or per-OS data dir), opens the DB, registers handlers, blocks on the MCP server.
  - [ ] Graceful shutdown on SIGINT/SIGTERM.
- [ ] Docs:
  - [ ] Bump CHANGELOG with the M1 entries.
  - [ ] Update README "Install" section to remove the "not working yet" warning.
  - [ ] Mark M1 row of the roadmap as 🟢 done.

### M1 risks / open questions

| # | Question                                                                     | Plan to resolve                                                       |
|---|------------------------------------------------------------------------------|------------------------------------------------------------------------|
| 1 | Where does `THOUGHTLINE_HOME` default to on Windows?                          | Use `%LOCALAPPDATA%/thoughtline/` (matches `os.UserCacheDir()`).        |
| 2 | Should `revision_count` be 0 or 1 for the first save?                         | Engram uses 0; mirror that for compatibility.                           |
| 3 | What does `tl_save` return when content equals the existing row exactly?     | Return `action: noop` with the existing id. Avoids spurious updates.    |
| 4 | UUIDv7 library?                                                               | Pick `github.com/google/uuid` v1.6+ (has Version 7) — already common.   |

---

## Pre-publish TODOs

Items the maintainer must do before / right after pushing the repo to GitHub. Ordered.

### Before first commit

- [x] ~~**Decide the GitHub URL.**~~ Repo is `github.com/AgusLoza2021/Thoughtline`. Updated in:
  - [x] ~~[`go.mod`](../go.mod) — `module` line~~
  - [x] ~~[`README.md`](../README.md) — install command, clone command~~
  - [x] ~~[`CHANGELOG.md`](../CHANGELOG.md) — `[Unreleased]` compare URL~~
  - [x] ~~[`CONTRIBUTING.md`](../CONTRIBUTING.md) — clone command~~
- [ ] **LICENSE copyright**: currently `Copyright (c) 2026 Thoughtline contributors`. Adjust to your name (e.g. `Copyright (c) 2026 Agustín Lozano`) if you want personal authorship credit. The Engram attribution paragraph at the bottom must stay.

### Before going public

- [ ] **Verify Engram attribution**. The LICENSE and README both credit Engram and link to it. Don't strip these — it's the right thing to do and keeps the door open for cross-pollination.
- [ ] **Replace the relative `_engram-research/` paths** in `docs/research/*.md` with permalinks to specific Engram commits on GitHub (e.g. `https://github.com/Gentleman-Programming/engram/blob/<sha>/internal/store/store.go#L789`). Right now they point at the local clone — fine for you, broken for anyone else.
- [ ] **Add a SECURITY.md** if accepting issues from the public (one paragraph: how to report security issues, expected response time).
- [ ] **Optional but nice**: GitHub issue templates (`bug.yml`, `feature.yml`) under `.github/ISSUE_TEMPLATE/`.
- [ ] **Optional**: a `CODE_OF_CONDUCT.md`. We've leaned on a one-liner in CONTRIBUTING for now; bring in a full document if the project grows.

### First push checklist

The repo is already initialized locally (`Desktop/thoughtline/.git`) with one auto-generated `Initial commit` that contains only `.gitattributes`. The remote has not been added yet. From the project root:

```bash
git add .
git commit -m "chore: M0 bootstrap — skeleton, docs, ADRs, taxonomy"
git remote add origin https://github.com/AgusLoza2021/Thoughtline.git
git push -u origin main
```

After pushing:

- [ ] Enable GitHub Actions for the repo (CI is already wired in `.github/workflows/ci.yml`).
- [ ] Tag `v0.0.1` once M1 is done — that's the first version we publish to `pkg.go.dev`.

---

## How to update this file

After every working session:

1. Move the **What got done this session** content to a `## YYYY-MM-DD — milestone-name` history section above this one (keeping `## Current state` at the top).
2. Update the M1 (or active milestone) checklist with what got checked off.
3. Add any new open questions to the table.
4. Strike-through items in the pre-publish TODO list as they're completed.

This file is a living artifact. If it goes stale, the rest of the docs lose their anchor.
