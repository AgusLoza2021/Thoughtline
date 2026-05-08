# Cognitive Config Specification

> Change: `brain-foundation`
> Status: shipped
> Operation: ADDED (new capability — no prior spec exists)

## Purpose

Defines the two-level configuration model: a singleton `global_config` row holding system-wide
defaults, and per-brain `config_json` holding only field overrides. Effective config is computed
as `deep-merge(global, brain_overrides)`. Covers schema, default values, tolerant parsing, and
propagation rules.

---

## Requirements

### Requirement: Global Config Singleton

The system MUST maintain a `global_config` table with a single row (`id = 1`). This row MUST
be created (with compiled-in defaults) during the v4 migration if it does not already exist.
The row MUST NOT be deleted — only updated.

#### Scenario: Global config row exists after migration

- GIVEN a fresh v4 database
- WHEN the migration completes
- THEN exactly one row exists in `global_config` with `id = 1` and valid default values

---

### Requirement: Config Schema — Sections and Defaults

The config MUST define these four top-level sections with the following default values:

**`ranking`** — weights that sum to 1.0 (compile-time validation SHOULD warn if they don't)

| Field | Default |
|---|---|
| `bm25` | 0.40 |
| `semantic` | 0.25 |
| `recency` | 0.15 |
| `importance` | 0.15 |
| `access` | 0.05 |

**`decay`**

| Field | Default |
|---|---|
| `half_life_days` | 30 |
| `floor` | 0.1 |

**`graph`**

| Field | Default |
|---|---|
| `auto_link_on_contradiction` | false |
| `auto_link_threshold` | 0.85 |
| `max_neighbors_per_node` | 50 |

**`simulation`**

| Field | Default |
|---|---|
| `enabled` | false |
| `synthetic_decay` | 1.0 |
| `time_multiplier` | 1.0 |
| `allow_destructive` | false |

#### Scenario: Brain with empty config_json gets global defaults

- GIVEN a brain B with `config_json = "{}"`
- WHEN `EffectiveConfig(B)` is called
- THEN the returned config matches the global defaults exactly (e.g. `ranking.bm25 = 0.40`)

---

### Requirement: Per-Brain Config Override (Partial JSON)

Each brain's `config_json` MUST hold ONLY the fields that differ from the global config. The
effective config for a brain is computed as `deep-merge(global_config, brain.config_json)` where
brain values override matching paths and all other paths fall through to the global value.

#### Scenario: Brain overrides a single decay field

- GIVEN global config has `decay.half_life_days = 30`
- AND brain B has `config_json = {"decay": {"half_life_days": 1}}`
- WHEN `EffectiveConfig(B)` is called
- THEN `decay.half_life_days = 1` (brain override)
- AND all other fields (e.g. `ranking.bm25 = 0.40`) come from the global config unchanged

#### Scenario: Brain override survives global change

- GIVEN brain B explicitly overrides `ranking.bm25 = 0.60`
- WHEN global config is updated to `ranking.bm25 = 0.50`
- THEN `EffectiveConfig(B).ranking.bm25` remains `0.60` (brain override wins)

---

### Requirement: Global Config Propagation

When a field in `global_config` is updated, all brains whose `config_json` does NOT explicitly
override that field MUST reflect the new global value immediately on the next `EffectiveConfig`
call. No re-save of individual brain rows is required — the merge is computed at read time.

#### Scenario: Global change propagates to non-overriding brain

- GIVEN brain B has `config_json = "{}"`
- AND global config has `ranking.bm25 = 0.40`
- WHEN global config is updated to `ranking.bm25 = 0.50`
- THEN `EffectiveConfig(B).ranking.bm25 = 0.50` immediately

---

### Requirement: Tolerant JSON Parsing (Forward Compatibility)

The system MUST parse `config_json` and global config blobs tolerantly: unknown fields MUST be
ignored (with a warning logged), never treated as errors. Missing fields MUST fall back to the
global value, then to the compiled default. This enables old binaries to read configs written
by newer versions without panicking.

#### Scenario: Unknown field in config_json is ignored

- GIVEN brain B has `config_json = {"experimental": {"foo": true}, "decay": {"floor": 0.2}}`
- WHEN `EffectiveConfig(B)` is called
- THEN no error is returned; a warning is logged about the unknown field `experimental.foo`
- AND `decay.floor = 0.2` (from brain override); all other fields from global defaults
