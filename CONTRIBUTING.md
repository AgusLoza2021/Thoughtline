# Contributing to Thoughtline

Thanks for considering a contribution. Thoughtline is small, opinionated, and we'd like to keep it that way — but well-scoped PRs that improve clarity, correctness, or coverage are very welcome.

## Before opening a PR

1. **Check `docs/PROGRESS.md`** to see which milestone is active. Work that pulls forward future milestones is fine, but flag it in the PR so we can sequence it.
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

## Code conventions

- **Package layout**: `cmd/thoughtline` (entry), `internal/server` (MCP wiring), `internal/storage` (SQLite), `internal/memory` (domain types). The split is deliberate — keep it.
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

## Code of conduct

Be kind, be precise, cite source lines. Disagreement is fine; condescension isn't.

## License

By contributing you agree your work is released under Thoughtline's [MIT License](LICENSE).
