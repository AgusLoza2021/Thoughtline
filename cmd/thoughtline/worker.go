package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// runWorker implements the `thoughtline worker` subcommand. It runs a
// one-shot retention janitor: archives pending rows older than retentionDur,
// then hard-deletes archived rows older than hardDeleteDur.
//
// The worker is designed to be invoked from a cron job or Task Scheduler. It
// is safe to run concurrently — SQLite's busy_timeout (set at Open) handles
// lock contention between two simultaneous runs gracefully.
func runWorker(ctx context.Context, args []string, errW io.Writer) error {
	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
	fs.SetOutput(errW)

	retentionStr := fs.String("retention", "168h", "soft-archive window (default 7 days)")
	hardDeleteStr := fs.String("hard-delete", "720h", "hard-delete window (default 30 days)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	retentionDur, err := time.ParseDuration(*retentionStr)
	if err != nil {
		return fmt.Errorf("parse --retention: %w", err)
	}
	hardDeleteDur, err := time.ParseDuration(*hardDeleteStr)
	if err != nil {
		return fmt.Errorf("parse --hard-delete: %w", err)
	}

	dbPath, err := resolveDBPath()
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}

	st, err := storage.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = st.Close() }()

	result, err := st.SweepPending(ctx, retentionDur, hardDeleteDur, time.Now())
	if err != nil {
		return fmt.Errorf("sweep: %w", err)
	}

	fmt.Fprintf(errW, "thoughtline worker: archived=%d deleted=%d\n",
		result.Archived, result.Deleted)
	return nil
}
