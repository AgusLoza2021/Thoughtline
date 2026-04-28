// Package main is the entry point for the Thoughtline MCP server.
//
// Thoughtline speaks the Model Context Protocol over stdio, exposing a set of
// tools that an MCP-compatible client (Claude Code, Cursor, Zed, ...) can call
// to persist and retrieve project memory tailored to a game-development
// workflow.
//
// Status: skeleton. The binary boots and exits cleanly. Tool registration
// (tl_save, tl_search, tl_context, ...) lands in milestone M1 — see
// docs/PROGRESS.md.
package main

import (
	"context"
	"fmt"
	"os"
)

// version is overwritten at build time via -ldflags "-X main.version=...".
var version = "0.0.0-dev"

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "thoughtline: %v\n", err)
		os.Exit(1)
	}
}

func run(_ context.Context) error {
	// Intentionally empty for the bootstrap milestone. The next session wires
	// the MCP server (mark3labs/mcp-go), the storage layer (modernc.org/sqlite),
	// and the first real tool (tl_save). See docs/ARCHITECTURE.md for the
	// target shape.
	fmt.Fprintf(os.Stderr, "thoughtline %s — skeleton build, no tools registered yet\n", version)
	return nil
}
