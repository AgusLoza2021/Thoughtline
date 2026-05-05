# cmd/migrate — Engram → Thoughtline migration tool

This binary reads your existing [Engram](https://github.com/Gentleman-Programming/engram) memory database and copies every active observation into Thoughtline. It is a **one-shot, idempotent migration tool** — safe to run multiple times, safe to interrupt and re-run.

---

## What this does

`migrate` opens your Engram database read-only, reads every observation that has not been soft-deleted, maps the column schema and type vocabulary to Thoughtline's format, and writes each row through `storage.Save()` — the same path the MCP server uses. This means FTS5 full-text indexing, normalized content hashing, and all validation rules fire exactly as they would for a normal `tl_save` call.

Your Engram `sync_id` values are preserved 1:1 so you can cross-reference migrated memories by their original ID.

---

## Before you start

1. **Back up your Engram database**:
   ```powershell
   Compress-Archive -Path "$env:USERPROFILE\.engram\*" -DestinationPath "$env:USERPROFILE\.engram-backup-$(Get-Date -Format 'yyyy-MM-dd').zip" -Force
   ```
   Verify the zip exists and is non-zero before proceeding. The migration never writes to Engram, but having a backup is non-negotiable.

2. **Verify both databases are reachable**:
   - Engram: `$env:USERPROFILE\.engram\engram.db` (default)
   - Thoughtline: `$env:LOCALAPPDATA\thoughtline\thoughtline.db` (default)

3. **Build the binary** (see next section).

---

## How to run

### Step 1 — Build

```bash
# From the Thoughtline repo root:
go build -o migrate.exe ./cmd/migrate/cmd
```

### Step 2 — Dry run first

Always run with `--dry-run` before writing anything. This reads and maps every row without touching the destination:

```bash
./migrate.exe --dry-run --verbose
```

Example output:
```
migrate: DRY RUN — no writes will be made
migrate: source=C:\Users\..\.engram\engram.db dest=C:\...\thoughtline.db log=...

migrate summary:
  migrated:                 0
  skipped (already exists): 0
  skipped (topic collision):0
  failed:                   0
  truncations:              0
  total processed:          291
  duration:                 42ms
```

Check `total processed` matches your expected Engram row count (soft-deleted rows are excluded automatically).

### Step 3 — Run for real

If the dry run shows `failed: 0` (or an acceptable number), run without `--dry-run`:

```bash
./migrate.exe
```

The tool prints a summary to stdout and writes a detailed per-row log to `%LOCALAPPDATA%\thoughtline\migrate-YYYY-MM-DD-HHMMSS.log`.

### Step 4 — Verify

Open `thoughtline ui` and check the Stats panel. The new total should equal:

```
pre-migration Thoughtline count + migrate summary "migrated" count
```

---

## Understanding the output

| Counter | Meaning |
|---------|---------|
| `migrated` | Rows successfully written to Thoughtline |
| `skipped (already exists)` | Rows whose `sync_id` was already in Thoughtline — safe on re-runs |
| `skipped (topic collision)` | Rows where `(project, topic_key)` already exists in Thoughtline as a native row — skipped to protect your data |
| `failed` | Rows that could not be migrated (oversized content, corrupt timestamps). Check the log file. |
| `truncations` | Rows whose title was > 200 characters — title was truncated, row was still migrated |
| `total processed` | Active (non-deleted) Engram rows seen by the migrator |

---

## Type mapping table

Engram uses a general-purpose type vocabulary. Thoughtline uses a gamedev-specific one. The mapping is deterministic:

| Engram type    | Thoughtline type | Provenance tag added  |
|----------------|------------------|-----------------------|
| `bugfix`       | `bugfix`         | none                  |
| `preference`   | `preference`     | none (scope forced to `personal`) |
| `decision`     | `decision`       | none                  |
| `architecture` | `architecture`   | none                  |
| `pattern`      | `convention`     | `origin-type:pattern` |
| `config`       | `convention`     | `origin-type:config`  |
| `discovery`    | `convention`     | `origin-type:discovery` |
| `manual`       | `convention`     | `origin-type:manual`  |
| *(any other)*  | `convention`     | `origin-type:<original>` |

The `origin-type:*` tag lets you find coerced rows later and re-classify them if you want. You can search for them in Thoughtline with `tl_search` filtering by tag.

---

## What to do on errors

**`failed: N` is not zero**

Open the log file at `%LOCALAPPDATA%\thoughtline\migrate-*.log` and look for `action=error` lines. Each has a `reason=` field explaining what went wrong.

Common causes:
- **Oversized content** (`content too large`): The row has more than 64 KiB of content. Thoughtline enforces this limit. To migrate the row manually, trim the content and use `tl_save` directly.
- **Corrupt timestamp** (`parse timestamp`): The `created_at` or `updated_at` field isn't valid ISO 8601. This is rare in Engram databases but possible in very old rows.
- **Unknown scope** (`unknown scope`): The row has a scope value other than `project` or `personal`. Inspect the raw row in Engram and fix before re-running.

**`skipped (topic collision): N` is unexpectedly high**

This means Thoughtline already has rows with the same `(project, topic_key)` as some Engram rows. The migrator skips those rows to protect your existing Thoughtline data. If you want to overwrite them, soft-delete the existing Thoughtline rows first via `tl_delete`, then re-run the migrator.

**Rollback**

If you need to undo the migration:
1. In Thoughtline, migrated rows have `sync_id` values that match Engram's original `sync_id`s (formatted like `obs-xxxxxxxx`). You can find and soft-delete them manually or write a short script.
2. Your Engram database is unaffected — the migrator never writes to it.
3. Your `.zip` backup lets you restore Engram to pre-migration state if needed.

---

## Re-running safely

The migrator is **idempotent**. If you run it twice:

- Rows that were already migrated appear as `skipped (already exists)` — they are identified by `sync_id`.
- No data is overwritten or duplicated.
- Rows that errored on the first run will be retried on the second run (they were never written, so they have no `sync_id` in the destination).

It is safe to interrupt the migrator mid-run and restart it.

---

## Log file location

Every run writes a structured log to:

```
%LOCALAPPDATA%\thoughtline\migrate-YYYY-MM-DD-HHMMSS.log
```

Format: one line per event, `key=value` pairs:

```
level=INFO msg=row_result sync_id=obs-abc123 action=created
level=WARN msg=title_truncated sync_id=obs-xyz789 original="215 runes"
level=INFO msg=row_result sync_id=obs-def456 action=skipped-duplicate reason=sync_id_already_exists
```

This is your audit trail. Keep it until you have verified the migration is correct.
