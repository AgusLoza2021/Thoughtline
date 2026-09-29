# ADR 0008 — The Engine Is the Product

**Status**: Accepted
**Supersedes**: [ADR 0007 — This Repository Owns the Vocabulary, Not the Tool Mechanics](0007-vocabulary-not-mechanics.md)
**Date**: 2026-09-29
**Evidence**: this repository at `dfdfecd`, re-read from `main`; the `tl_*` surface, the
SQLite store and the tests that cover both

---

## Context

ADR 0007 recorded a decision made when the project had one deliverable left. The MCP
server had been retired, its task was handed to [Engram](https://github.com/Gentleman-Programming/engram),
and what remained here was a memory *vocabulary* — with the intent that the words would be
the product and the mechanics would belong to someone else.

Everything in 0007 follows from that premise, and is correct given it: a repository that
prints no mechanics cannot disagree with them; a type list nobody enforces is a
recommendation and must be described as one; a claim about another project's behaviour has
to cite that project at a named revision or it is not evidence.

The premise is what changed, and it did not change because anyone decided to change it. The
engine was never removed. `cmd/thoughtline` still builds an MCP server over stdio;
`internal/server` still exposes a `tl_*` tool surface; `internal/storage` is still SQLite
with FTS5 search; `internal/memory` is still the domain layer those tools validate through;
`internal/dashboard` and `cmd/migrate` are still here. All of it compiles, and all of it
passes.

What `cd93aa1` removed was the **distribution**: the release workflow, the Claude Code
plugin files, and the prose that described the server. A commit deleted the paperwork and
left the machine running, and for two days the repository described itself as something its
own source contradicted. The front door sent readers to a different project; the server it
shipped was one `go build` away, and the plugin that registered it was restored a day later.

ADR 0007's own amendment is the sharpest evidence that the premise had failed. It is dated
2026-09-29 and it retieres the catalogue to fourteen types, adds a `global` scope and
permits a dot in `topic_key` — justified entirely by *the other engine's* behaviour, because
only that engine was thought to be live. The same amendment had to note that twelve of those
fourteen types had no review horizon in it. A product whose rules are written to fit
somebody else's engine, and are not enforced by anybody's, is not a product.

## Decision

**1. This repository ships and maintains the memory server.** The `tl_*` MCP server is the
deliverable, along with the vocabulary that shapes what goes into it. The distribution that
was withdrawn on `cd93aa1` is restored: the release workflow, the Claude Code plugin
manifest, and the pages that teach the integration. A repository that ships a binary and
tells nobody how to run it has not shipped anything.

**2. This repository owns the mechanics of its own tool surface, and states them.** ADR
0007 §1 forbade restating another project's mechanics, for a good reason: a hand-maintained
copy of somebody else's interface has no mechanism behind it. That reason does not transfer.
Our tool surface is our own, it lives in this repository, and its source of truth is the
code that implements it — not a pinned reading of a commit we do not control. Where a page
describes a tool, the tool's own schema wins, and a change to the schema is a change to the
page.

**3. The type vocabulary is enforced, and the enforcement is code.** ADR 0007 §4 held that
the type list was ours and a **recommendation**, because Engram validates the field nowhere
and folds an unknown value to `manual`. That is no longer the situation: `tl_save` routes
every write through `internal/memory.Validate`, which rejects a type outside
`AllTypes()` with `ErrInvalidType`, rejects a scope outside `ScopeProject`/`ScopePersonal`,
and requires a `preference` to be `personal` — the opposite coupling, with its own sentinel
error. The vocabulary is a gate now. A type the catalogue lists and the validator rejects is
a bug in one of the two, and it is not a bug this repository can describe as a guideline.

The same rule covers the storage shape. `tags` is a real column with its own format check,
so a page that tells the agent to keep its tags on a `**Tags**:` line is teaching a
convention on top of a field that exists. Whether that convention should survive is a
product question with a code answer available, and it is not settled by leaving the page
ambiguous.

**4. Claims about another project are still read at that project's revision.** ADR 0007 §3
stands unchanged and unweakened, because it was never about this project's fate. Where this
repository states what Engram does — and it still does, in the migrator's documentation and
in the comparison — the statement cites Engram at a commit, and a claim that cannot be
re-derived there does not belong here.

**5. Behaviour is stated once.** ADR 0007 §2 stands. `docs/AGENT-SETUP.md` holds the
canonical drop-in block; the per-editor guides carry a paste-ready block of their own and
name the canonical one. A guide is allowed to be short. It is not allowed to be a second
copy of something that changes on its own schedule.

**6. The catalogue and the validator must agree, and the resolution is a decision, not a
tidy-up.** `docs/design/memory-domain.md` lists fourteen types in two tiers, documents a
`global` scope, and permits a dot in `topic_key` for version numbers. `internal/memory`
closes the set at eleven, accepts two scopes, and rejects a dotted key. Under 0007 this was
a documented divergence against an engine nobody maintained. Under this record it is a
promise the shipped tool breaks: `tl_save(type: "discovery")` fails. Which side moves is the
owner's call — the catalogue was grown by measurement, and the validator was frozen by a
retirement — and it is recorded here as an open, named defect rather than resolved by
whichever file happens to be edited first.

## Consequences

**The drift class changes direction.** Under 0007 the risk was restating somebody else's
mechanics. The risk now is that a page and the code disagree, and the mechanism against it
is the ordinary one: the code is the source of truth, the tests are the check, and the
documents are corrected against them. That is weaker than a generated page and much stronger
than a note asking everyone to keep two copies in step — which is the mechanism 0007
correctly rejected.

**Retirement is not rewritten out of the record.** The commits that retired the project, the
CHANGELOG entries that announce it, and the ADR 0007 text itself all stay. A project that
comes back and deletes the evidence that it left cannot be audited by anyone who arrives
later, and the reversal is the interesting part. 0007 is marked superseded and otherwise
untouched.

**Two engines exist and the tension is accepted, not solved.** The catalogue's rules were
written to describe the ecosystem this project shares, and its own server now enforces a
different subset of them. Converging the schemas is explicitly not the goal — the shared
vocabulary is the reason the two can be compared at all. The goal is that this repository's
own pages describe this repository's own server, and that where the catalogue reaches
further than the validator, it says so.

**A release becomes possible again, and cutting one is a separate decision.** Restoring the
workflow makes publishing available; it does not publish. The version question — whether the
next tag is `v0.2.0` or a re-cut of `v0.1.0` — belongs to the owner, and the tip has to be
finished before either answer is safe, since `main` is mid-feature.

**The review cost of one filename is real.** Restoring the surface puts `SECURITY.md` back in
the changed set, and that name alone routes a candidate to the higher review tier. It is
recorded as a finding rather than worked around by renaming a file to please a heuristic.

## Alternatives considered

**Keep ADR 0007 and re-scope it to the vocabulary only.** Rejected: the decision it records
is that the vocabulary is *not* enforced, and that is the specific claim the code now
contradicts. An ADR whose central premise is false is not rescued by narrowing its title.

**Delete ADR 0007 and write this one in its place.** Rejected: 0007 is an accurate record of
what was decided on 2026-09-28 and why, and its evidence — the revision of Engram it read,
the four stale clauses it found, the measurement across 2,336 observations — is still the
best account of that period. Supersession is what the format is for.

**Settle the fourteen-versus-eleven question by shrinking the catalogue to the validator.**
Rejected as an unforced decision. The three extra types were added by measurement and are
the most-used types in the ecosystem this project shares; the closure at eleven is an
artifact of the retirement. Growing the validator is the more likely answer, but it changes
what the shipped tool accepts, and that is not a change to make inside a documentation
sweep.

**Leave the whole question open and describe the divergence in a comment only.** Rejected:
the comment is worth writing, and it was written — `internal/memory`'s package doc now
states the disagreement. But a comment is not a decision, and a shipped tool that rejects a
documented input will surprise a user before it surprises a maintainer.
