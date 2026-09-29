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
vocabulary (which of the eleven types, where tags go, how `topic_key` is shaped) and the
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
Eleven types tuned to gamedev work is the project's reason to exist. Engram does not
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

**Known limitation, left open.** `decayReviewAfterMonths` (`internal/store/store.go:380-384`)
assigns a review horizon to exactly three `type` strings — `decision` (six months), `policy`
(twelve) and `preference` (three) — and the comment above the map states the consequence
itself: *"Types absent from this map get `review_after` = NULL (Phase 1 behavior)."* Nine of
this vocabulary's eleven types therefore have no review horizon and never appear in `mem_review`.
The options are (a) ask upstream to extend that map, (b) make the canonical `type` one of
Engram's values and move gamedev specificity into `topic_key` and tags, or (c) accept the gap
and lean on `mem_doctor`. This record does not choose: (b) is the only option fully under this
project's control, but it spends the vocabulary's specificity, which is the differentiator —
so it is the owner's call, not this record's.

**Scope.** Every capability named here is local to one machine. No networked or multi-user
behaviour is documented by this decision, and none is implied by it.

## Alternatives considered

**Restate Engram's whole tool surface.** Rejected: twenty-odd tool descriptions copied out
of someone else's repository is the failure being corrected, at larger scale. It converts
a small accurate page into a large stale one.

**Delete the protocol page and send readers to Engram.** Rejected: the vocabulary and the
behaviour it implies are the product. Engram's protocol cannot teach which of eleven types
a Unity pipeline gotcha belongs in.

**Keep the duplicate block and add a "keep these in agreement" note.** Rejected: a note is not a
mechanism. The note would have been written by the same change that let the clause go stale
in seven files — it had no such note and did not need one to drift.
