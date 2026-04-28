# Changelog

All notable changes to Thoughtline will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) starting at the M1 release.

## [Unreleased]

### Added
- Project skeleton: `cmd/thoughtline`, `internal/{server,storage,memory}` with package docs.
- README, LICENSE (MIT, with attribution to Engram), `.gitignore`, `go.mod` targeting Go 1.25.
- Architectural reconnaissance of [Engram](https://github.com/Gentleman-Programming/engram) — three deep-dive docs in `docs/research/`.
- ADR 0001: Architecture baseline — copy Engram's pattern set, customize taxonomy.
- ADR 0002: Search strategy — FTS5 + BM25 in v1, embeddings reserved for M5.
- Memory taxonomy design (`docs/design/memory-domain.md`) with gamedev-first types.
- Architecture overview (`docs/ARCHITECTURE.md`) with mermaid diagrams.
- Example MCP client config in `examples/mcp-config.example.json`.
- Basic CI workflow: `go vet` and `go build` on every push.
- CONTRIBUTING.md with PR and ADR conventions.

### Status
- Bootstrap milestone (M0) complete. M1 (`tl_save`) is next.

[Unreleased]: https://github.com/AgusLoza2021/Thoughtline/compare/HEAD...HEAD
