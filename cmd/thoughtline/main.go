// Package main is the entry point for the Thoughtline MCP server.
//
// Thoughtline speaks the Model Context Protocol over stdio. An
// MCP-compatible client (Claude Code, Cursor, Zed, ...) launches this binary
// and exchanges JSON-RPC messages over stdin/stdout. All tool implementations
// live in internal/server; storage in internal/storage.
//
// Configuration (env vars):
//
//	THOUGHTLINE_HOME   Directory for the SQLite database. Defaults to the
//	                   platform user-data dir (see ADR 0003):
//	                     - Linux:   $XDG_DATA_HOME or ~/.local/share, then /thoughtline
//	                     - macOS:   ~/Library/Application Support/thoughtline
//	                     - Windows: %LOCALAPPDATA%\thoughtline
//	                   On first launch after upgrading from a build that used
//	                   the cache dir, the DB is auto-migrated.
//	THOUGHTLINE_DB     Override the database file path entirely. Wins over
//	                   THOUGHTLINE_HOME if set. Useful for tests.
//	THOUGHTLINE_PROJECT  Override the default project identifier. Defaults to
//	                   the basename of the working directory at startup.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/AgusLoza2021/Thoughtline/internal/dashboard"
	"github.com/AgusLoza2021/Thoughtline/internal/server"
	"github.com/AgusLoza2021/Thoughtline/internal/storage"
)

// version is overwritten at build time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Subcommand routing. With no args we default to the MCP stdio server
	// (the path an MCP client takes when it spawns this binary).
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	var err error
	switch cmd {
	case "", "serve":
		err = runServer(ctx)
	case "ui", "dashboard":
		err = runDashboard(ctx, os.Args[2:])
	case "protocol":
		err = runProtocol(os.Args[2:])
	case "hook":
		err = runHook(ctx, os.Args[2:], os.Stdin, os.Stderr)
	case "worker":
		err = runWorker(ctx, os.Args[2:], os.Stderr)
	case "version", "-v", "--version":
		fmt.Printf("thoughtline %s\n", version)
		return
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "thoughtline: unknown subcommand %q\n\n", cmd)
		printUsage()
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "thoughtline: %v\n", err)
		os.Exit(1)
	}
}

// printUsage prints a short subcommand reference. Kept tight on purpose —
// the README is the comprehensive reference.
func printUsage() {
	fmt.Fprintln(os.Stderr, `Thoughtline — local-first persistent memory for AI assistants.

Usage:
  thoughtline              run the MCP stdio server (default; what your AI client launches)
  thoughtline serve        same as no-arg invocation
  thoughtline ui           open the interactive dashboard (TUI)
                           flags: --theme {brand|zbrush|mono}, --no-splash,
                                  --no-update-check, --splash-ms N
  thoughtline protocol     emit the active-protocol markdown to stdout
                           flags: --event {session-start|post-compaction},
                                  --project NAME, -o FILE
  thoughtline version      print the binary version and exit
  thoughtline help         print this help and exit

Environment:
  THOUGHTLINE_HOME    directory for the SQLite database
  THOUGHTLINE_DB      override the database file path entirely
  THOUGHTLINE_PROJECT default project identifier (otherwise auto-detected from cwd)

See README.md for the full reference and per-tool examples.`)
}

func runServer(ctx context.Context) error {
	dbPath, err := resolveDBPath()
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}

	st, err := storage.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open storage at %s: %w", dbPath, err)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "thoughtline: close storage: %v\n", cerr)
		}
	}()

	cfg := server.Config{
		Version:        version,
		DefaultProject: resolveDefaultProject(),
	}

	srv := server.New(st, cfg)

	fmt.Fprintf(os.Stderr,
		"thoughtline %s — db=%s default-project=%q\n",
		version, dbPath, cfg.DefaultProject,
	)

	// ServeStdio blocks until the client disconnects (EOF on stdin) or
	// the context is cancelled. Either is a clean exit.
	if err := mcpserver.ServeStdio(srv, mcpserver.WithStdioContextFunc(func(_ context.Context) context.Context {
		return ctx
	})); err != nil {
		// EOF on stdin is the normal way the client signals shutdown.
		// Treat it as success.
		if isCleanShutdown(err) {
			return nil
		}
		return fmt.Errorf("serve stdio: %w", err)
	}
	return nil
}

func runDashboard(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	themeName := fs.String("theme", "brand", "color palette: brand | zbrush | mono")
	noSplash := fs.Bool("no-splash", false, "skip the intro splash screen")
	noUpdateCheck := fs.Bool("no-update-check", false, "skip the GitHub release lookup")
	splashMS := fs.Int("splash-ms", 1500, "splash duration in milliseconds")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dbPath, err := resolveDBPath()
	if err != nil {
		return fmt.Errorf("resolve db path: %w", err)
	}

	st, err := storage.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open storage at %s: %w", dbPath, err)
	}
	defer func() {
		if cerr := st.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "thoughtline: close storage: %v\n", cerr)
		}
	}()

	cfg := dashboard.Config{
		Version:        version,
		DBPath:         dbPath,
		Project:        resolveDefaultProject(),
		ThemeName:      *themeName,
		Splash:         !*noSplash,
		SplashDuration: time.Duration(*splashMS) * time.Millisecond,
		CheckUpdates:   !*noUpdateCheck,
	}
	return dashboard.Run(ctx, st, cfg)
}

// resolveDBPath returns the SQLite file path to open. Order of precedence:
//
//	1. THOUGHTLINE_DB (full file path)
//	2. THOUGHTLINE_HOME/thoughtline.db
//	3. <user data dir>/thoughtline/thoughtline.db (see ADR 0003)
//
// On the default-path branch, a one-time migration runs from the legacy
// cache-dir location used by builds before ADR 0003. The parent directory
// is created if missing. Migration is bypassed entirely when either
// override is set.
func resolveDBPath() (string, error) {
	if explicit := os.Getenv("THOUGHTLINE_DB"); explicit != "" {
		if err := os.MkdirAll(filepath.Dir(explicit), 0o755); err != nil {
			return "", err
		}
		return explicit, nil
	}

	if home := os.Getenv("THOUGHTLINE_HOME"); home != "" {
		if err := os.MkdirAll(home, 0o755); err != nil {
			return "", fmt.Errorf("create home %s: %w", home, err)
		}
		return filepath.Join(home, "thoughtline.db"), nil
	}

	base, err := dataDir()
	if err != nil {
		return "", fmt.Errorf("user data dir: %w", err)
	}
	home := filepath.Join(base, "thoughtline")

	if err := migrateLegacyCacheDir(home); err != nil {
		// Migration is best-effort. Log and continue with the new path so
		// the binary always starts; users with weird permissions on the old
		// path can still launch and see the empty new DB.
		fmt.Fprintf(os.Stderr, "thoughtline: legacy DB migration skipped: %v\n", err)
	}

	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", fmt.Errorf("create home %s: %w", home, err)
	}
	return filepath.Join(home, "thoughtline.db"), nil
}

// dataDir returns the platform-specific user data directory. Unlike
// os.UserCacheDir, this points at a location OS cleanup tools will not wipe.
// See ADR 0003.
func dataDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// LocalAppData is non-roaming, machine-local, and is the standard
		// place for application data on Windows. Same value os.UserCacheDir
		// returns here, but we read it explicitly so the call site reads as
		// "data" not "cache".
		if v := os.Getenv("LocalAppData"); v != "" {
			return v, nil
		}
		return os.UserCacheDir()
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support"), nil
	default: // linux, BSDs, plan9, ...
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return v, nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share"), nil
	}
}

// migrateLegacyCacheDir performs the one-time move from the pre-ADR-0003
// cache-dir location to the new data-dir location. No-op when the source
// doesn't exist, the destination already does, or both resolve to the same
// path (Windows, where LocalAppData is both cache and data).
func migrateLegacyCacheDir(newHome string) error {
	cacheBase, err := os.UserCacheDir()
	if err != nil {
		// No cache dir resolvable -> nothing to migrate.
		return nil
	}
	oldHome := filepath.Join(cacheBase, "thoughtline")

	if filepath.Clean(oldHome) == filepath.Clean(newHome) {
		return nil
	}

	oldDB := filepath.Join(oldHome, "thoughtline.db")
	if _, err := os.Stat(oldDB); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat legacy DB: %w", err)
	}

	newDB := filepath.Join(newHome, "thoughtline.db")
	if _, err := os.Stat(newDB); err == nil {
		// Already migrated, or user populated the new path manually. Don't
		// overwrite real data; leave the old one alone so the user decides.
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(newHome), 0o755); err != nil {
		return fmt.Errorf("create parent of %s: %w", newHome, err)
	}

	if err := os.Rename(oldHome, newHome); err != nil {
		// Cross-device rename or other failure. Don't try copy+delete here:
		// too easy to half-finish and leave the user in an ambiguous state.
		return fmt.Errorf("rename %s -> %s: %w", oldHome, newHome, err)
	}

	fmt.Fprintf(os.Stderr,
		"thoughtline: migrated DB from legacy cache path %q to data path %q (one-time, see ADR 0003)\n",
		oldHome, newHome,
	)
	return nil
}

// resolveDefaultProject returns the project identifier used when a tool call
// omits `project`. Order: $THOUGHTLINE_PROJECT, basename of cwd, "default".
func resolveDefaultProject() string {
	if p := os.Getenv("THOUGHTLINE_PROJECT"); p != "" {
		return p
	}
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return "default"
	}
	base := filepath.Base(cwd)
	if base == "." || base == string(filepath.Separator) {
		return "default"
	}
	return base
}

// isCleanShutdown reports whether err corresponds to the MCP client closing
// its end of the pipe — the normal exit path for a stdio MCP server.
func isCleanShutdown(err error) bool {
	if err == nil {
		return true
	}
	msg := err.Error()
	for _, marker := range []string{"EOF", "file already closed", "context canceled"} {
		if containsCI(msg, marker) {
			return true
		}
	}
	return false
}

func containsCI(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if eqFold(haystack[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func eqFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
