# Memory Graph Specification

> Change: `brain-foundation`
> Status: shipped
> Operation: ADDED (new capability — no prior spec exists)

## Purpose

Defines the `memory_links` table: typed directed edges between memories within a brain.
Covers the closed relation enum, brain-scope enforcement (no cross-brain links), self-loop
prevention, uniqueness rules, cascade behavior, and the query surface (`Neighbors`, `Subgraph`).

---

## Requirements

### Requirement: MemoryLink Entity Shape

A `MemoryLink` record MUST have the following fields:

| Field | Type | Constraints |
|---|---|---|
| `id` | INTEGER | PRIMARY KEY AUTOINCREMENT |
| `brain_id` | INTEGER | NOT NULL, FK → `brains.id` ON DELETE CASCADE |
| `from_id` | INTEGER | NOT NULL, FK → `memories.id` ON DELETE CASCADE |
| `to_id` | INTEGER | NOT NULL, FK → `memories.id` ON DELETE CASCADE |
| `relation` | TEXT | NOT NULL — CHECK (closed enum, see Requirement: Relation Enum) |
| `weight` | REAL | NOT NULL DEFAULT 1.0 |
| `source` | TEXT | NOT NULL — CHECK(`manual`, `auto`, `imported`) |
| `note` | TEXT | NULLABLE |
| `created_at` | INTEGER | NOT NULL — unix epoch ms |

The database MUST enforce a UNIQUE constraint on `(from_id, to_id, relation)`.

#### Scenario: Creating a valid same-brain link succeeds

- GIVEN memories M1 and M2 both belong to brain B
- WHEN a link is created with `from_id=M1`, `to_id=M2`, `relation="supersedes"`, `source="manual"`
- THEN the link is persisted with an assigned `id` and `weight=1.0`

---

### Requirement: Relation Enum

The `relation` field MUST accept exactly these values (case-sensitive):

`supersedes`, `contradicts`, `refines`, `depends_on`, `references`, `related`, `derived_from`

The database MUST enforce this via a CHECK constraint. The application MUST also validate before writing.

#### Scenario: Known relation value accepted

- GIVEN `relation = "contradicts"`
- WHEN a link is saved
- THEN the row is persisted

#### Scenario: Unknown relation value rejected

- GIVEN `relation = "inspired_by"` (not in the closed set)
- WHEN a link is saved
- THEN storage returns an error; no row is inserted

---

### Requirement: Brain-Scope Enforcement (No Cross-Brain Links)

The system MUST reject any link where `from_id` and `to_id` belong to different brains. The check MUST happen at the storage layer, not only in the calling code.

#### Scenario: Cross-brain link is rejected with explicit error

- GIVEN memory M1 belongs to brain A and memory M2 belongs to brain B (A ≠ B)
- WHEN a link with `from_id=M1`, `to_id=M2` is created
- THEN storage returns an explicit "cross-brain link" error; no row is inserted

#### Scenario: Same-brain link is accepted

- GIVEN M1 and M2 both belong to brain A
- WHEN a link is created between them
- THEN the link is persisted successfully

---

### Requirement: Self-Loop Prevention

The system MUST reject any link where `from_id == to_id`.

#### Scenario: Self-loop is rejected

- GIVEN memory M1 in brain A
- WHEN a link is created with `from_id=M1` and `to_id=M1`
- THEN storage returns a "self-loop" error; no row is inserted

---

### Requirement: Uniqueness of (from_id, to_id, relation)

The same `(from_id, to_id, relation)` triple MUST NOT appear more than once. The same pair MAY have multiple links with different relation values.

#### Scenario: Duplicate triple is rejected

- GIVEN a link (M1 → M2, "refines") already exists
- WHEN another link (M1 → M2, "refines") is created
- THEN storage returns a uniqueness error; no second row is inserted

#### Scenario: Same pair with different relation is allowed

- GIVEN a link (M1 → M2, "refines") already exists
- WHEN a link (M1 → M2, "related") is created
- THEN both links exist; no error

---

### Requirement: Cascade Deletion

Deleting a brain MUST CASCADE delete all its `memory_links` rows. Deleting a memory MUST CASCADE delete all incident links (both inbound and outbound) for that memory.

#### Scenario: Deleting a brain cascades to its links

- GIVEN brain B has 5 links between its memories
- WHEN brain B is deleted
- THEN all 5 links are deleted; no orphaned `memory_links` rows reference the deleted brain

#### Scenario: Deleting a memory cascades to incident links

- GIVEN memory M1 has 3 outbound links and 2 inbound links
- WHEN memory M1 is deleted
- THEN all 5 incident links are deleted; no orphaned rows remain

---

### Requirement: Neighbors Query

The system MUST provide a `Neighbors(memID, relation?)` query that returns all incident links
for the given memory — both inbound (`to_id = memID`) and outbound (`from_id = memID`).
When `relation` is supplied, only links matching that relation value are returned.

#### Scenario: Neighbors returns inbound and outbound links

- GIVEN M1 has 2 outbound links and 1 inbound link
- WHEN `Neighbors(M1)` is called with no relation filter
- THEN 3 links are returned (both directions)

#### Scenario: Neighbors filtered by relation

- GIVEN M1 has 1 link with `relation="supersedes"` and 2 with `relation="related"`
- WHEN `Neighbors(M1, relation="supersedes")` is called
- THEN exactly 1 link is returned

---

### Requirement: Subgraph Query

The system MUST provide a `Subgraph(brainID, filters?)` query that returns a `{nodes []Memory, edges []MemoryLink}` result containing only data within the requested brain. Optional filters MAY narrow the result by relation type or memory IDs.

#### Scenario: Subgraph returns only its brain's data

- GIVEN brain A has 4 memories and 3 links; brain B has 2 memories and 1 link
- WHEN `Subgraph(brainID=A)` is called
- THEN the result contains exactly 4 nodes and 3 edges; no brain B data is included

#### Scenario: Subgraph with relation filter narrows edges

- GIVEN brain A has 3 links: 2 "supersedes" and 1 "related"
- WHEN `Subgraph(brainID=A, relation="supersedes")` is called
- THEN the result contains 2 edges; the "related" link is excluded
