// Command migrate performs a one-shot migration from Engram's SQLite database
// to Thoughtline. It reads Engram observations, maps types and columns per the
// engram-migration spec, and writes through Thoughtline's storage.Save() so
// FTS5 triggers, normalized hashes, and validation all fire correctly.
//
// Usage:
//
//	migrate [--source PATH] [--dest PATH] [--dry-run] [--verbose]
//
// Flags:
//
//	--source   Path to engram.db (default: %USERPROFILE%\.engram\engram.db on Windows)
//	--dest     Path to thoughtline.db (default: %LOCALAPPDATA%\thoughtline\thoughtline.db)
//	--dry-run  Read and map rows without writing; prints what would happen
//	--verbose  Print each row result as it is processed
//
// Exit codes:
//
//	0  All rows processed without errors (or dry-run completed)
//	1  One or more rows failed (see log file and stdout summary for details)
//	2  Fatal error (DB unreachable, log file unwritable, etc.)
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	migrate "github.com/AgusLoza2021/Thoughtline/cmd/migrate"
)

func main() {
	src := flag.String("source", defaultEngramPath(), "Path to engram.db (read-only source)")
	dest := flag.String("dest", defaultThoughtlinePath(), "Path to thoughtline.db (destination)")
	dryRun := flag.Bool("dry-run", false, "Map rows without writing to destination")
	verbose := flag.Bool("verbose", false, "Print each row result during migration")
	flag.Parse()

	cfg := migrate.Config{
		Source:  *src,
		Dest:    *dest,
		DryRun:  *dryRun,
		Verbose: *verbose,
	}

	// Open (or create) the log file.
	logPath, logFile, err := openLogFile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: cannot open log file: %v\n", err)
		os.Exit(2)
	}
	defer func() { _ = logFile.Close() }()

	logWriter := io.MultiWriter(os.Stdout, logFile)
	logger := migrate.NewStructuredLogger(logWriter)

	if cfg.DryRun {
		fmt.Fprintf(os.Stderr, "migrate: DRY RUN — no writes will be made\n")
	}
	fmt.Fprintf(os.Stderr, "migrate: source=%s dest=%s log=%s\n", cfg.Source, cfg.Dest, logPath)

	start := time.Now()
	ctx := context.Background()

	_ = logger // logger is passed to Run in a future refactor; currently Run uses NopLogger internally
	summary, err := migrate.Run(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: fatal: %v\n", err)
		os.Exit(2)
	}

	duration := time.Since(start)

	// Print per-row results when --verbose.
	if cfg.Verbose {
		for _, r := range summary.Rows {
			logger.Log("INFO", "row result", map[string]string{
				"sync_id": r.SyncID,
				"action":  r.Action,
				"reason":  r.Reason,
			})
		}
	}

	// Final summary to stdout.
	fmt.Printf("\nmigrate summary:\n")
	fmt.Printf("  migrated:                 %d\n", summary.Created)
	fmt.Printf("  skipped (already exists): %d\n", summary.SkippedDuplicate)
	fmt.Printf("  skipped (topic collision):%d\n", summary.SkippedTopicCol)
	fmt.Printf("  failed:                   %d\n", summary.Errors)
	fmt.Printf("  truncations:              %d\n", summary.Truncations)
	fmt.Printf("  total processed:          %d\n", summary.Total)
	fmt.Printf("  duration:                 %s\n", duration.Round(time.Millisecond))

	if summary.Errors > 0 {
		fmt.Fprintf(os.Stderr, "\nmigrate: %d row(s) failed — check log: %s\n", summary.Errors, logPath)
		os.Exit(1)
	}
}

// defaultEngramPath returns the default Engram DB path for this OS.
func defaultEngramPath() string {
	home := os.Getenv("USERPROFILE")
	if home == "" {
		home = os.Getenv("HOME")
	}
	return filepath.Join(home, ".engram", "engram.db")
}

// defaultThoughtlinePath returns the default Thoughtline DB path for this OS.
func defaultThoughtlinePath() string {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		// macOS / Linux fallback.
		base, err := os.UserCacheDir()
		if err != nil {
			base = filepath.Join(os.Getenv("HOME"), ".cache")
		}
		local = base
	}
	return filepath.Join(local, "thoughtline", "thoughtline.db")
}

// openLogFile creates the Thoughtline log directory if needed and opens a
// timestamped log file. Returns the path and an open *os.File.
func openLogFile() (string, *os.File, error) {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			base = filepath.Join(os.Getenv("HOME"), ".cache")
		}
		local = base
	}
	logDir := filepath.Join(local, "thoughtline")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", nil, fmt.Errorf("create log dir %s: %w", logDir, err)
	}

	ts := time.Now().Format("2006-01-02-150405")
	logPath := filepath.Join(logDir, "migrate-"+ts+".log")
	f, err := os.Create(logPath)
	if err != nil {
		return "", nil, fmt.Errorf("create log file %s: %w", logPath, err)
	}
	return logPath, f, nil
}
