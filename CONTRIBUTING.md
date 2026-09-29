# Contributing to Thoughtline

Thanks for considering a contribution. Thoughtline is small, opinionated, and we'd like to keep it that way — but well-scoped PRs that improve clarity, correctness, or coverage are very welcome.

## Before opening a PR

1. **Check the open issues.** There is no active-milestone document any more: the v0.1.0 engine's roadmap (`docs/PROGRESS.md`) is retired along with the engine, and the live work is the vocabulary. Work that pulls a future idea forward is fine, but flag it in the PR so we can sequence it.
2. **Read the relevant ADR.** If your change touches architecture (search strategy, storage backend, MCP shape), there's likely an ADR explaining the prior decision. New architectural choices need a new ADR.
3. **Open an issue first for non-trivial work.** A 30-line bugfix can land directly. A new memory type, a new tool, a schema migration — open an issue first.

## Local setup

```bash
git clone https://github.com/AgusLoza2021/Thoughtline.git
cd Thoughtline
go build ./cmd/thoughtline
go vet ./...
```

The default storage path is `$THOUGHTLINE_HOME/thoughtline.db`, falling back to a per-OS data directory.

> **The Go engine is archived, and it is not the product.** The live part of this repository is the vocabulary: [`docs/design/memory-domain.md`](docs/design/memory-domain.md) (the type catalogue, and the only complete copy of it) and [`presets/AGENTS.md`](presets/AGENTS.md) (what your agent reads). [`internal/memory/`](internal/memory/) is the archived engine's copy of the same list. Building the engine still works, and it is worth doing if you are changing its historical record — but a change that only touches the vocabulary needs no Go build at all.

## Code conventions

- **Package layout**: `cmd/thoughtline` (entry), `internal/server` (MCP wiring), `internal/storage` (SQLite), `internal/memory` (domain types). The split is deliberate — keep it. Note that the type catalogue is no longer wired to this code: it lives in [`docs/design/memory-domain.md`](docs/design/memory-domain.md), and `internal/memory` holds the archived engine's old copy of it.
- **No CGO**. We use `modernc.org/sqlite` precisely to keep cross-compilation trivial. PRs that introduce CGO will be asked to revert.
- **Error wrapping**: use `fmt.Errorf("...: %w", err)`. Surface root causes; don't swallow them.
- **Logs to stderr only**. stdout is reserved for MCP protocol traffic.
- **Tests**: every storage function gets a table-driven test. Tool handlers get an integration test that exercises the full request/response over an in-memory MCP transport when feasible.

## ADRs (Architecture Decision Records)

ADRs live in [`docs/decisions/`](docs/decisions/) and follow [Michael Nygard's format](https://github.com/joelparkerhenderson/architecture-decision-record/blob/main/locales/en/templates/decision-record-template-by-michael-nygard/index.md):

- **Title**: `NNNN-short-imperative-headline.md`
- **Status**: Proposed / Accepted / Superseded
- **Context** / **Decision** / **Consequences**

A change is "ADR-worthy" if reverting it would require touching the schema, the MCP tool surface, or the public CLI flags.

## Commit messages

Conventional commits, no emojis, no AI co-author lines.

```
feat(storage): add FTS5 contentless virtual table
fix(server): graceful shutdown on SIGTERM
docs(adr): record decision to defer embeddings to M5
chore(ci): bump golangci-lint to v1.65
```

## Attribution and copied code

Thoughtline is MIT-licensed. Take attribution seriously: if you bring code into this
repository from somewhere else, name the project, its licence, and what you took.

The passive-capture design was informed by
[`claude-mem`](https://github.com/thedotmack/claude-mem) (by thedotmack) — the hook list
and the queue-then-promote shape. No code, prompt text or schema definitions were copied;
what was borrowed is the architectural idea.

**This section used to be an "AGPL firewall."** It declared claude-mem to be AGPL-3.0,
listed five forbidden strings, required every PR touching `internal/pending/`,
`cmd/thoughtline/hook.go` or `cmd/thoughtline/worker.go` to carry a one-line affirmation,
and a CI guard enforced all of it. **The licence claim was false** — claude-mem has never
been AGPL, and its current licence is Apache-2.0. The guard and the affirmation were
removed rather than reworded, because the false premise was their only reason to exist.
See [`docs/COMPARISON.md`](docs/COMPARISON.md) for the full record, and
[`docs/decisions/0004-passive-capture-via-hooks.md`](docs/decisions/0004-passive-capture-via-hooks.md)
for the amendment to the decision that cited it.

## Code of conduct

Be kind, be precise, cite source lines. Disagreement is fine; condescension isn't.

## License

By contributing you agree your work is released under Thoughtline's [MIT License](LICENSE).
