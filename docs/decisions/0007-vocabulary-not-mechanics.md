# ADR 0007 — This Repository Owns the Vocabulary, Not the Tool Mechanics

**Status**: Accepted
**Date**: 2026-09-28
**Evidence revision**: Engram `3ba7df6235f5a58ca0898dc312421881c1fb0acf` (2026-09-28)

---

## Context

The project changed shape. What shipped as `thoughtline` — an MCP server with its
own `tl_*` tools — was retired, and what remains is a memory vocabulary for gamedev
work that runs on [Engram](https://github.com/Gentleman-Programming/engram). The
[README](../../README.md) states the division: Engram owns storage, search and the
tool surface; this repository owns the words.

One page did not survive that change intact.
[`docs/integrations/claude-code-protocol.md`](../integrations/claude-code-protocol.md)
opened with the claim that it is "the canonical Thoughtline memory protocol block for
AI assistants". That was true while the tools it described were ours. After the pivot
the tools became Engram's, the page went on describing them from memory, and nothing
in this repository forced the copy to agree with the original. The same is true of the
32-line drop-in block that lived byte-identical in [`AGENT-SETUP.md`](../AGENT-SETUP.md)
and in all six per-editor guides.

That is a structural failure, not a proofreading failure. A copy of another project's
interface, maintained by hand, has no mechanism behind it. By 2026-09-28 the copied
mechanics were wrong four times over:

- `scope` was described as having two values. Engram's `normalizeScope`
  (`internal/store/store.go:11532`) accepts `personal` and `global` and folds everything
  else to `project`. The stale clause was byte-identical in seven files.
- `type` was treated as a constraint. Engram validates it nowhere; its own tool
  description suggests a different set of values and its default is `manual`.
- Search was taught in the wrong order. Engram's canonical order is `mem_context`, then
  `mem_search`, then `mem_get_observation`.
- Search results were undersold as "snippet and metadata". They carry `state` and the
  relation annotations, and those are meant to be acted on.

Engram's own documentation states the rule that was broken, and states it in its
authority section: *"'must change together' means review for behavioral alignment, not
copy identical text into every host."*

## Decision

**1. This repository does not restate another project's tool mechanics.**
`docs/integrations/claude-code-protocol.md` teaches what this repository owns — the
vocabulary (which of the catalogued types, where tags go, how `topic_key` is shaped) and the
behaviour that follows from it (when to save, when to correct a memory instead of saving
it twice, what to pin, how to treat a recalled memory that is stale or contested, how to
close a session and leave learnings behind). For mechanics it points at Engram's canonical
protocol rather than reprinting it:
<https://github.com/Gentleman-Programming/engram/blob/main/DOCS.md#memory-protocol-full-text>

**2. Behaviour is stated once.** [`docs/AGENT-SETUP.md`](../AGENT-SETUP.md) holds the
canonical drop-in block. The six per-editor guides keep a paste-ready block too, but only
of the vocabulary, and each one names the canonical block and says why it is not repeated
there. A guide is allowed to be short; it is not allowed to be a second copy.

**3. Claims about Engram are read from Engram, at a named revision.** This record rests on
`3ba7df6235f5a58ca0898dc312421881c1fb0acf`, the `main` of 2026-09-28. A claim about
Engram's behaviour that cannot be re-derived at a named revision does not belong in this
repository — the previous pinned reading was `3687c2f`, and the difference between the
two is exactly the kind of drift a citation is supposed to expose.

**4. The `type` vocabulary remains this repository's, and it is a recommendation.**
The type list is the project's reason to exist. Engram does not
enforce them, and this repository will not pretend otherwise: the guidance is to always
pass a type, not a claim that the server checks one.

## Consequences

**The drift class is closed for mechanics.** A page that prints no mechanics cannot
disagree with them. When Engram changes, the work is to re-read Engram at a revision — a
bounded job — instead of auditing seven hand-maintained copies of a block.

**The vocabulary is still duplicated, and deliberately.** The type list, the Tags line and
the `topic_key` shape appear in eight places. Those are ours, they change on our schedule,
and a reader pasting a block into an editor needs them present. What was removed is the
duplication of things we do not control.

**Decided: the vocabulary stays, and the request goes upstream.** `decayReviewAfterMonths`
(`internal/store/store.go:380-384`) assigns a review horizon to exactly three `type` strings —
`decision` (six months), `policy` (twelve) and `preference` (three) — and the comment above the
map states the consequence itself: *"Types absent from this map get `review_after` = NULL
(Phase 1 behavior)."* Twelve of this vocabulary's fourteen types therefore have no review horizon
and never appear in `mem_review`.

Three options were weighed. **(b) — make the canonical `type` one of Engram's values and move
gamedev specificity into `topic_key` and tags — is rejected:** it is the only option this
project can act on alone, and it spends the differentiator. Three consecutive revisions of this
repository have been about the type list being the product; re-cutting it to fit a provisional
mechanism in someone else's engine is backwards. **(c) — accept the gap and lean on
`mem_doctor` — is rejected on the facts:** `mem_doctor` runs ten checks
(`internal/diagnostic/checks.go`), and none of them reads `review_after`; it cannot see a stale
memory at all. What remains is **(a): keep
the vocabulary and ask upstream to extend the map — or better, to make the decay policy
configurable**, since a per-type horizon is a statement about *our* types and belongs with the
vocabulary rather than the engine.

The cost of waiting is bounded, and that was verified rather than assumed. Nothing breaks and no
capability is lost — and even where the horizon does exist it is Phase-1 coarse: it is written
once on insert (`internal/store/store.go:3608`, which notes it "runs only for NEW inserts (not
topic_key revisions or deduplication)"), no tool can set it (`UpdateObservation`,
`internal/store/store.go:4302`, accepts title, content, find/replace, type, scope and
`topic_key`), and the only other writer is `markReviewed`. So an actively-maintained `decision`
memory still ages into `needs_review` while a stale `perf-gotcha` never does. The signal is
coarse in both directions, which is a further reason not to redesign this vocabulary around it.

In the meantime the review tools are not the whole of memory hygiene. `mem_search`'s rendered
result line carries each memory's created date (`internal/mcp/mcp.go:1312`), so for the nine
types the engine does not cover, age is read from the result and judged by the agent. Revisit
this decision when Engram extends the map or makes it configurable; the change here would then
be one commit.

**Scope.** Every capability named here is local to one machine. No networked or multi-user
behaviour is documented by this decision, and none is implied by it.

## Alternatives considered

**Restate Engram's whole tool surface.** Rejected: twenty-odd tool descriptions copied out
of someone else's repository is the failure being corrected, at larger scale. It converts
a small accurate page into a large stale one.

**Delete the protocol page and send readers to Engram.** Rejected: the vocabulary and the
behaviour it implies are the product. Engram's protocol cannot teach which of the
catalogued types a Unity pipeline gotcha belongs in.

**Keep the duplicate block and add a "keep these in agreement" note.** Rejected: a note is not a
mechanism. The note would have been written by the same change that let the clause go stale
in seven files — it had no such note and did not need one to drift.

## Amendment — 2026-09-29 (the catalogue was retiered)

The counts written above are from 2026-09-28, and the shape of the list has changed since.
Measured across 2,336 observations in 13 projects, every agent session on this stack is
instructed to choose from `bugfix | decision | architecture | discovery | pattern | config |
preference` — a set this repository's catalogue denied three of. `discovery` alone accounts
for 447 observations, the most-chosen agent type in every project, while the seven game-dev
types added here total 4 uses across the same 2,336.

The catalogue is therefore now two tiers: the seven the ecosystem already teaches as core,
and the seven game-dev additions as optional extensions, fourteen in total. The list lives in
exactly one file, [`docs/design/memory-domain.md`](../design/memory-domain.md) — the previous
revision repeated it in fourteen files and was wrong in all of them by the time anyone
checked. `scope` now documents `global`, which has 30 measured uses, and `topic_key` permits
a dot, because all four dotted keys measured are version numbers.

Nothing in this decision changes. The vocabulary is still ours, still a recommendation, and
the request still goes upstream.
