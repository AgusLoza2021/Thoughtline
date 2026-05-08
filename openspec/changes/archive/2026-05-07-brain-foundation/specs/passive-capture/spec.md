# Delta for Passive Capture

> Change: `brain-foundation`
> Base spec: `openspec/specs/passive-capture/spec.md`

## ADDED Requirements

### Requirement: tl_promote Resolves brain_id from project

When `tl_promote` creates a memory from a pending event (base spec Requirement 6), the new
memory MUST have its `brain_id` populated. `tl_promote` MUST resolve the brain by matching
`pending_events.project` to `brains.slug`. If no matching brain is found, the promotion for
that event MUST fail with error `"no brain found for project: <value>"` — no orphaned memory
MUST be created without a valid `brain_id`.

#### Scenario: tl_promote sets brain_id on created memory

- GIVEN a `pending_event` with `project = "my-game"` and a brain with `slug = "my-game"` exists
- WHEN `tl_promote` is called for that event
- THEN the created memory has `brain_id` pointing to the "my-game" brain

#### Scenario: tl_promote fails if no matching brain

- GIVEN a `pending_event` with `project = "unknown-project"` and no brain with that slug exists
- WHEN `tl_promote` is called for that event
- THEN the event is NOT promoted; the response contains `{ status: "error", error: "no brain found for project: unknown-project" }`
