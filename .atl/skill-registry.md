# Skill Registry

**Delegator use only.** Any agent that launches sub-agents reads this registry to resolve compact rules, then injects them directly into sub-agent prompts. Sub-agents do NOT read this registry or individual SKILL.md files.

See `_shared/skill-resolver.md` for the full resolution protocol.

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| Writing Go tests, using teatest, or adding test coverage | go-testing | C:\Users\Agustin Lozano\.claude\skills\go-testing\SKILL.md |
| Creating games, building game engines, web-based games with HTML5/Canvas/WebGL | game-engine | C:\Users\Agustin Lozano\.claude\skills\game-engine\SKILL.md |
| Building, debugging, or optimizing Claude API / Anthropic SDK apps | claude-api | (built-in skill) |
| Ingesting 3D asset libraries (FBX/OBJ) into Unity, composing scenes | asset-pipeline | C:\Users\Agustin Lozano\.claude\skills\asset-pipeline\SKILL.md |
| Creating a pull request, opening a PR | branch-pr | C:\Users\Agustin Lozano\.claude\skills\branch-pr\SKILL.md |
| Creating a GitHub issue, reporting a bug | issue-creation | C:\Users\Agustin Lozano\.claude\skills\issue-creation\SKILL.md |
| Parallel adversarial review, blind judge review | judgment-day | C:\Users\Agustin Lozano\.claude\skills\judgment-day\SKILL.md |
| Creating a new skill, adding agent instructions | skill-creator | C:\Users\Agustin Lozano\.claude\skills\skill-creator\SKILL.md |

## Compact Rules

Pre-digested rules per skill. Delegators copy matching blocks into sub-agent prompts as `## Project Standards (auto-resolved)`.

### go-testing
- Use table-driven tests: `tests := []struct{ name, input string; wantErr bool }{...}` with `t.Run(tt.name, func(t *testing.T){...})`
- Colocate `*_test.go` in the same package as the code under test (not a separate `test/` dir)
- Use `t.TempDir()` for on-disk SQLite in tests — never `:memory:` (FTS5 edge cases differ)
- Use `t.Helper()` in test helper functions
- For Bubbletea TUI: test Model state transitions directly via `m.Update(msg)`; use `teatest.NewTestModel` for interactive flows
- For golden file tests: store in `testdata/` and use `-update` flag to regenerate
- Integration tests go in `integration_test.go` files; use `net/http/httptest` for HTTP layer
- Never skip cleanup: use `t.Cleanup(func() { ... })` for resource teardown

### branch-pr
- Always create an issue first before a PR (issue-first enforcement)
- PR title: imperative mood, under 70 chars, no trailing period
- PR body: Summary bullets + Test plan checklist

### judgment-day
- Launch two independent blind judge sub-agents in parallel
- Do NOT share judge outputs with each other before synthesis
- Synthesize both verdicts before returning to orchestrator

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| — | — | No project-level CLAUDE.md, AGENTS.md, or .cursorrules found |

> **Note**: Project conventions come from the global `~/.claude/CLAUDE.md`. Key rules relevant to this project:
> - Never build after changes (`go build` forbidden in agent runs)
> - Strict TDD Mode: enabled
> - Use Read/Glob/Grep tools — never cat/grep/find/sed/ls
> - Conventional commits only, no Co-Authored-By attribution
