// Package main is the entry point for the Thoughtline MCP server.
//
// Thoughtline speaks the Model Context Protocol over stdio. An
// MCP-compatible client (Claude Code, Cursor, Zed, ...) launches this binary
// and exchanges JSON-RPC messages over stdin/stdout. All tool implementations
// live in internal/server; storage in internal/storage.
//
// Configuration (env vars):
//
//	THOUGHTLINE_HOME   Directory for the SQLite database. Defaults to
//	                   $XDG_DATA_HOME/thoughtline (Linux), ~/Library/Application Support/thoughtline (macOS),
//	                   or %LOCALAPPDATA%\thoughtline (Windows).
//	THOUGHTLINE_DB     Override the database file path entirely. Wins over
//	                   THOUGHTLINE_HOME if set. Useful for tests.
//	THOUGHTLINE_PROJECT  Override the default project identifier. Defaults to
//	                   the basename of the working directory at startup.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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
		err = runDashboard(ctx)
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

func runDashboard(ctx context.Context) error {
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
		Version: version,
		DBPath:  dbPath,
		Project: resolveDefaultProject(),
	}
	return dashboard.Run(ctx, st, cfg)
}

// resolveDBPath returns the SQLite file path to open. Order of precedence:
//
//	1. THOUGHTLINE_DB (full file path)
//	2. THOUGHTLINE_HOME/thoughtline.db
//	3. <user cache dir>/thoughtline/thoughtline.db
//
// The parent directory is created if missing.
func resolveDBPath() (string, error) {
	if explicit := os.Getenv("THOUGHTLINE_DB"); explicit != "" {
		if err := os.MkdirAll(filepath.Dir(explicit), 0o755); err != nil {
			return "", err
		}
		return explicit, nil
	}

	home := os.Getenv("THOUGHTLINE_HOME")
	if home == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("user cache dir: %w", err)
		}
		home = filepath.Join(base, "thoughtline")
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		return "", fmt.Errorf("create home %s: %w", home, err)
	}
	return filepath.Join(home, "thoughtline.db"), nil
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
