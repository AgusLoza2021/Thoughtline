package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// Run executes the full migration from Config.Source (Engram DB) to
// Config.Dest (Thoughtline DB). It processes all active rows, accumulating
// results in a Summary. Individual row errors are non-fatal — Run always
// processes every row and returns the complete Summary regardless of how many
// rows errored.
//
// If Config.DryRun is true, Run reads and maps all rows but performs no writes
// to the destination. Counters in the returned Summary reflect what would have
// happened.
func Run(ctx context.Context, cfg Config) (Summary, error) {
	logger := NopLogger{}
	start := time.Now()

	// --- Open Engram DB (read-only) ---
	// Note: do NOT request WAL mode on a read-only DSN. WAL-init writes to
	// .db-shm on first connect, which fails on mode=ro. The user's real
	// engram.db already has WAL sidecars from prior engram runs, so opening
	// it read-only without forcing WAL is safe — SQLite picks up the existing
	// WAL automatically. Test fixtures don't have sidecars, so forcing WAL
	// here would break them with SQLITE_READONLY (8).
	engramDSN := "file:" + cfg.Source + "?mode=ro&_pragma=query_only(1)"
	engramDB, err := sql.Open("sqlite", engramDSN)
	if err != nil {
		return Summary{}, fmt.Errorf("run: open engram db %s: %w", cfg.Source, err)
	}
	defer func() { _ = engramDB.Close() }()

	if err := engramDB.PingContext(ctx); err != nil {
		return Summary{}, fmt.Errorf("run: ping engram db: %w", err)
	}

	// --- Open Thoughtline storage ---
	tlStore, err := storage.Open(ctx, cfg.Dest)
	if err != nil {
		return Summary{}, fmt.Errorf("run: open thoughtline storage %s: %w", cfg.Dest, err)
	}
	defer func() { _ = tlStore.Close() }()

	// Raw DB handle to the same file — needed for sync_id post-UPDATE.
	rawDB, err := sql.Open("sqlite",
		cfg.Dest+"?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return Summary{}, fmt.Errorf("run: open raw dest db: %w", err)
	}
	defer func() { _ = rawDB.Close() }()

	// --- Read source rows ---
	observations, err := ReadObservations(ctx, engramDB)
	if err != nil {
		return Summary{}, fmt.Errorf("run: read observations: %w", err)
	}

	var summary Summary
	summary.Total = len(observations)

	// --- Process rows ---
	for _, obs := range observations {
		m, mapErr := MapRow(obs, logger)
		if mapErr != nil {
			// Map errors: oversized content, bogus scope, unparseable timestamp.
			summary.Errors++
			summary.Rows = append(summary.Rows, RowResult{
				SyncID: obs.SyncID,
				Action: "error",
				Reason: mapErr.Error(),
			})
			continue
		}

		// Count title truncations: if the mapped title is shorter than the
		// source title (in rune count), a truncation happened.
		if runeLen(m.Title) < runeLen(obs.Title) {
			summary.Truncations++
		}

		if cfg.DryRun {
			// Dry-run: count what would have happened without writing.
			summary.Rows = append(summary.Rows, RowResult{
				SyncID: obs.SyncID,
				Action: "dry-run",
			})
			continue
		}

		result, writeErr := writeRow(ctx, tlStore, rawDB, m, obs.SyncID)
		if writeErr != nil {
			summary.Errors++
			summary.Rows = append(summary.Rows, RowResult{
				SyncID: obs.SyncID,
				Action: "error",
				Reason: writeErr.Error(),
			})
			continue
		}

		switch result.Action {
		case "created":
			summary.Created++
		case "skipped-duplicate":
			summary.SkippedDuplicate++
		case "skipped-topic-collision":
			summary.SkippedTopicCol++
		}
		summary.Rows = append(summary.Rows, result)
	}

	summary.Rows = filterDeletedFromSummary(summary.Rows)
	_ = start // duration is computed by main.go from elapsed time, not stored here

	return summary, nil
}

// runeLen returns the number of Unicode code points in s.
func runeLen(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}

// filterDeletedFromSummary is a no-op placeholder. Soft-deleted rows are
// filtered at the reader level (WHERE deleted_at IS NULL), so they never
// appear in the processing loop and are not present in Rows. The
// SkippedDeleted counter is always 0 via Run() — it exists in Summary for
// forward compatibility and for tests that build Summary manually.
func filterDeletedFromSummary(rows []RowResult) []RowResult { return rows }
